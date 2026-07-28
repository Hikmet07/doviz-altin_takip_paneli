package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/altinkaynak-bot/backend/internal/api"
	"github.com/altinkaynak-bot/backend/internal/db"
	"github.com/altinkaynak-bot/backend/internal/models"
	"github.com/altinkaynak-bot/backend/internal/websocket"
)

// Scheduler - Periyodik veri çekme ve temizleme görevlerini yöneten yapı
type Scheduler struct {
	apiClient     *api.Client      // Veri çekilecek API istemcisi
	repo          *db.Repository   // Veritabanı işlemleri için repository
	hub           *websocket.Hub   // WebSocket üzerinden yayın yapmak için hub
	fetchInterval time.Duration    // Veri çekme sıklığı
	retentionDays int              // Verilerin kaç gün saklanacağı
}

// NewScheduler - Yeni bir Scheduler örneği oluşturur
func NewScheduler(apiClient *api.Client, repo *db.Repository, hub *websocket.Hub, fetchInterval time.Duration, retentionDays int) *Scheduler {
	return &Scheduler{
		apiClient:     apiClient,
		repo:          repo,
		hub:           hub,
		fetchInterval: fetchInterval,
		retentionDays: retentionDays,
	}
}

// Start - Tüm periyodik görevleri (fetcher ve cleanup) başlatır
func (s *Scheduler) Start(ctx context.Context) {
	slog.Info("Scheduler başlatılıyor", "fetch_interval", s.fetchInterval)
	go s.startFetcher(ctx)
	go s.startCleanup(ctx)
}

// startFetcher - API'den veri çekme döngüsünü başlatır
// Her fetchInterval sürede bir çalışır
func (s *Scheduler) startFetcher(ctx context.Context) {
	// Başlangıçta verileri hemen çek
	s.fetchAndBroadcast(ctx)

	ticker := time.NewTicker(s.fetchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done(): // Context iptal edilirse durdur
			slog.Info("Veri çekme döngüsü durduruldu")
			return
		case <-ticker.C: // Her tick olduğunda verileri çek
			s.fetchAndBroadcast(ctx)
		}
	}
}

// startCleanup - Eski verileri temizleme döngüsünü başlatır
// Her gece saat 00:00'da çalışır
func (s *Scheduler) startCleanup(ctx context.Context) {
	for {
		// Gece yarısına kadar olan süreyi hesapla
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		duration := next.Sub(now)

		select {
		case <-ctx.Done(): // Context iptal edilirse durdur
			slog.Info("Temizleme döngüsü durduruldu")
			return
		case <-time.After(duration): // Gece yarısı olunca temizle
			deleted, err := s.repo.CleanupOldRecords(ctx, s.retentionDays)
			if err != nil {
				slog.Error("Eski kayıtlar temizlenemedi", "error", err)
			} else {
				slog.Info("Eski kayıtlar başarıyla temizlendi", "deleted_count", deleted)
			}
		}
	}
}

// fetchAndBroadcast - API'den veri çeker, DB'ye kaydeder, WebSocket ile yayınlar
func (s *Scheduler) fetchAndBroadcast(ctx context.Context) {
	// 1. API'den verileri çek
	records, err := s.apiClient.FetchAll(ctx)
	if err != nil {
		slog.Error("Veriler API'den çekilemedi", "error", err)
		return
	}

	// 2. Verileri veritabanına kaydet
	if err := s.repo.SaveRecords(ctx, records); err != nil {
		slog.Error("Veriler veritabanına kaydedilemedi", "error", err)
		return
	}

	// 2b. Eğer veritabanı henüz boşsa veya az kayıt varsa geçmiş 30 günlük örnek verileri doldur
	if err := s.repo.SeedHistoricalDataIfEmpty(ctx, records); err != nil {
		slog.Warn("Geçmiş veri tohumlama hatası", "error", err)
	}

	// 2c. Günlük fiyat farkı ve % değişimi 24 saat öncesinin verisiyle hesaplayıp ekle
	records = s.repo.AttachDailyChanges(ctx, records)

	// 3. Altın ve döviz kategorilerini ayır
	var pd models.PriceData
	for _, r := range records {
		if r.Kategori == "currency" {
			pd.Currency = append(pd.Currency, r)
		} else if r.Kategori == "gold" {
			pd.Gold = append(pd.Gold, r)
		}
	}

	// 4. WebSocket mesajı oluştur
	msg := models.WSMessage{
		Type:      "update",
		Data:      pd,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// 5. Mesajı tüm client'lara yayınla
	if err := s.hub.BroadcastMessage(msg); err != nil {
		slog.Error("WebSocket yayını yapılamadı", "error", err)
	}

	// 6. Başarı durumunu logla
	slog.Info("Fiyatlar güncellendi ve yayınlandı", "record_count", len(records))
}
