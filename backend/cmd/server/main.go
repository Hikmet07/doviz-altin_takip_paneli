package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/altinkaynak-bot/backend/internal/api"
	"github.com/altinkaynak-bot/backend/internal/config"
	"github.com/altinkaynak-bot/backend/internal/db"
	"github.com/altinkaynak-bot/backend/internal/handler"
	"github.com/altinkaynak-bot/backend/internal/models"
	"github.com/altinkaynak-bot/backend/internal/scheduler"
	"github.com/altinkaynak-bot/backend/internal/websocket"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	// 1. Loglama yapılandırması - JSON formatında yapılandırılmış (structured) loglama
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	slog.Info("Uygulama başlatılıyor...")

	// 2. Yapılandırma dosyasını yükle
	cfg := config.Load()

	// 3. Veritabanı bağlantısını oluştur
	database, err := db.NewDatabase(cfg.DBPath)
	if err != nil {
		slog.Error("Veritabanı başlatılamadı", "error", err)
		os.Exit(1)
	}
	// Uygulama kapanırken veritabanı bağlantısını kapat
	defer database.Close()

	// 4. Veritabanı işlemlerini yönetecek repository oluştur
	repo := db.NewRepository(database)

	// 5. Dış sistemden veri çekecek API istemcisini oluştur
	apiClient := api.NewClient()

	// 6. WebSocket hub oluştur ve başlat
	hub := websocket.NewHub()
	go hub.Run()

	// 7. WebSocket üzerinden gelen mesajları dinleyen callback ayarla
	// HistoryRequest tipindeki istekleri karşılar ve geçmiş verileri döndürür
	hub.SetOnMessage(func(data []byte) {
		var req models.HistoryRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return
		}
		if req.Type == "history_request" {
			days := req.Days
			if days <= 0 {
				days = 30 // Varsayılan olarak 30 günlük veri
			}
			
			// Veritabanından istenen aracın geçmiş verilerini al
			records, err := repo.GetHistory(context.Background(), req.Kod, days)
			if err != nil {
				slog.Error("Geçmiş veriler alınamadı", "error", err)
				return
			}
			
			// WebSocket istemcisine geçmiş verileri döndür
			msg := models.WSMessage{
				Type:      "history",
				Kod:       req.Kod,
				Data:      records,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			hub.BroadcastMessage(msg)
		}
	})

	// Global context oluştur ve sinyalleri yakala
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 8. Zamanlayıcıyı (scheduler) oluştur ve başlat
	// Veri çekme ve eski verileri temizleme işlemlerini yönetir
	sched := scheduler.NewScheduler(apiClient, repo, hub, cfg.FetchInterval, cfg.CleanupRetentionDays)
	sched.Start(ctx)

	// 9. HTTP isteklerini karşılayacak handler'ı oluştur
	h := handler.NewHandler(repo, hub)

	// 10. Router oluştur ve HTTP rotalarını kaydet
	router := mux.NewRouter()
	h.RegisterRoutes(router)

	// 11. CORS ayarlarını yapılandır
	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})
	
	// CORS middleware'ini router'a uygula
	handlerWithCors := c.Handler(router)

	// 12. HTTP sunucusunu yapılandır
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: handlerWithCors,
		// Timeout ayarları güvenli bir sunucu için önemlidir
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 13. Sunucuyu ayrı bir goroutine'de başlat
	go func() {
		slog.Info("HTTP Sunucusu başlatıldı", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Sunucu hatası", "error", err)
			os.Exit(1)
		}
	}()

	// 14. İşletim sistemi sinyallerini dinle (SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	// Sinyal gelene kadar bekle
	<-quit
	slog.Info("Sunucu kapatılıyor...")

	// 15. Graceful shutdown (Nazik kapanış) için 10 saniyelik timeout belirle
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Scheduler ve diğer arka plan işlemlerini iptal et
	cancel()

	// Mevcut isteklerin tamamlanmasını bekle ve sunucuyu kapat
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Sunucu kapatılırken hata oluştu", "error", err)
	}

	slog.Info("Sunucu güvenli bir şekilde kapatıldı")
}
