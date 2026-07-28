package db

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/altinkaynak-bot/backend/internal/models"
)

// Repository - Veritabanı CRUD işlemlerini yöneten yapı
// Veritabanı ile olan tüm okuma ve yazma işlemlerini tek bir merkezde toplar.
type Repository struct {
	// db - İşlemlerin yürütüleceği Database nesnesi
	db *Database
}

// NewRepository - Yeni bir Repository oluşturur
// db: Veritabanı bağlantılarını yönetecek nesne
func NewRepository(db *Database) *Repository {
	return &Repository{
		db: db,
	}
}

// scanner - Satırları okuyabilen ortak bir arayüz
// sql.Row ve sql.Rows nesnelerinin her ikisi de bu arayüzü sağlar
type scanner interface {
	Scan(dest ...interface{}) error
}

// scanRecord - Veritabanı satırından PriceRecord nesnesini okur
// Tekrar eden kodları azaltmak için kullanılan yardımcı bir fonksiyondur
// Sütun sırası: id, kod, aciklama, alis, satis, kategori, guncelleme_zamani, created_at
func scanRecord(row scanner) (models.PriceRecord, error) {
	var record models.PriceRecord
	// guncelleme_zamani ve created_at string olarak okunacak (SQLite'da DATETIME aslında TEXT)
	var guncellenmeStr, createdAtStr string

	// Sütunları sırasıyla PriceRecord nesnesinin alanlarına eşleştir
	err := row.Scan(
		&record.ID,
		&record.Kod,
		&record.Aciklama,
		&record.Alis,
		&record.Satis,
		&record.Kategori,
		&guncellenmeStr,
		&createdAtStr,
	)
	if err != nil {
		return record, fmt.Errorf("kayıt okunamadı: %w", err)
	}

	// Zaman string'lerini time.Time'a çevir
	// SQLite'da depolanan format: "2006-01-02T15:04:05Z" veya "2006-01-02 15:04:05"
	record.GuncellenmeZamani = parseTimeString(guncellenmeStr)
	record.CreatedAt = parseTimeString(createdAtStr)

	return record, nil
}

// parseTimeString - SQLite'dan okunan çeşitli zaman formatlarını time.Time'a çevirir
// SQLite farklı formatlarda zaman saklayabilir, bu fonksiyon hepsini destekler
func parseTimeString(s string) time.Time {
	// Desteklenen formatları sırayla dene
	formats := []string{
		time.RFC3339,              // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05Z",   // UTC açık belirteç
		"2006-01-02 15:04:05",    // SQLite varsayılan formatı
		"2006-01-02T15:04:05",    // T ayracıyla
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	// Hiçbir format uyuşmazsa sıfır değer döndür ve logla
	slog.Warn("Zaman formatı çözümlenemedi", "değer", s)
	return time.Time{}
}

// SaveRecords - Fiyat kayıtlarını toplu olarak veritabanına kaydeder
// Aynı kod ve güncelleme zamanına sahip kayıtları tekrar eklemez (duplikasyon kontrolü)
// Transaction kullanarak atomik kayıt sağlar — ya hepsi eklenir ya hiçbiri
func (r *Repository) SaveRecords(ctx context.Context, records []models.PriceRecord) error {
	// Yeni bir veritabanı işlemi (transaction) başlatıyoruz
	tx, err := r.db.DB().BeginTx(ctx, nil)
	if err != nil {
		slog.Error("Transaction başlatılamadı", "hata", err)
		return fmt.Errorf("transaction başlatılamadı: %w", err)
	}

	// İşlem bittiğinde, eğer hata varsa geri al (rollback)
	// Commit başarılı olursa rollback no-op olur
	defer tx.Rollback()

	// Duplikasyon kontrolü için kullanılacak hazır SQL sorgusu (Prepared Statement)
	// Aynı enstrüman koduna ve güncelleme zamanına sahip kayıt var mı kontrol eder
	checkStmt, err := tx.PrepareContext(ctx, `
		SELECT COUNT(*) FROM price_records 
		WHERE kod = ? AND guncelleme_zamani = ?
	`)
	if err != nil {
		return fmt.Errorf("kontrol sorgusu hazırlanamadı: %w", err)
	}
	defer checkStmt.Close()

	// Kayıt ekleme işlemi için kullanılacak hazır SQL sorgusu
	// created_at sütunu UTC olarak kaydedilir (SQLite datetime karşılaştırmaları için tutarlılık)
	insertStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO price_records (kod, aciklama, alis, satis, kategori, guncelleme_zamani, created_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("ekleme sorgusu hazırlanamadı: %w", err)
	}
	defer insertStmt.Close()

	eklenenKayitSayisi := 0

	// Gelen tüm kayıtları dön
	for _, record := range records {
		var count int
		// Güncelleme zamanını SQLite uyumlu string formatına çevir
		guncellenmeStr := record.GuncellenmeZamani.UTC().Format("2006-01-02 15:04:05")

		// Veritabanında aynı koda ve güncelleme zamanına sahip bir kayıt olup olmadığını kontrol et
		err := checkStmt.QueryRowContext(ctx, record.Kod, guncellenmeStr).Scan(&count)
		if err != nil {
			return fmt.Errorf("kayıt kontrolü başarısız: %w", err)
		}

		// Eğer kayıt daha önce eklenmemişse (count == 0) ekleme işlemini gerçekleştir
		if count == 0 {
			// created_at'ı da UTC olarak kaydet — datetime('now') ile tutarlı olması için
			createdAtStr := time.Now().UTC().Format("2006-01-02 15:04:05")
			_, err = insertStmt.ExecContext(
				ctx,
				record.Kod,
				record.Aciklama,
				record.Alis,
				record.Satis,
				record.Kategori,
				guncellenmeStr,
				createdAtStr,
			)
			if err != nil {
				return fmt.Errorf("kayıt eklenemedi (%s): %w", record.Kod, err)
			}
			eklenenKayitSayisi++
		}
	}

	// Tüm işlemler başarılıysa veritabanına yansıt (commit yap)
	if err = tx.Commit(); err != nil {
		slog.Error("Transaction commit başarısız", "hata", err)
		return fmt.Errorf("transaction commit başarısız: %w", err)
	}

	slog.Info("Kayıtlar başarıyla kaydedildi", "toplam_gelen", len(records), "yeni_eklenen", eklenenKayitSayisi)
	return nil
}

// AttachDailyChanges - Her bir enstrüman kaydına sabah borsa açılışındaki fiyatla karşılaştırarak günlük fark ve % değişim değerlerini ekler
func (r *Repository) AttachDailyChanges(ctx context.Context, records []models.PriceRecord) []models.PriceRecord {
	// Bugünün başlangıcı (00:00:00 UTC / Sabah açılış verisi)
	startOfDay := time.Now().UTC().Format("2006-01-02 00:00:00")

	// Bugünün ilk kaydını (sabah açılış fiyatını) getiren sorgu
	stmtTodayOpen, err := r.db.DB().PrepareContext(ctx, `
		SELECT satis FROM price_records 
		WHERE kod = ? AND created_at >= ? 
		ORDER BY created_at ASC LIMIT 1
	`)
	if err != nil {
		slog.Warn("Günlük açılış sorgusu hazırlanamadı", "hata", err)
		return records
	}
	defer stmtTodayOpen.Close()

	// Eğer bugünün verisi henüz yoksa en son dünün kapanış kaydını getiren sorgu
	stmtPrevDayClose, err := r.db.DB().PrepareContext(ctx, `
		SELECT satis FROM price_records 
		WHERE kod = ? AND created_at < ? 
		ORDER BY created_at DESC LIMIT 1
	`)
	if err != nil {
		slog.Warn("Dünün kapanış sorgusu hazırlanamadı", "hata", err)
		return records
	}
	defer stmtPrevDayClose.Close()

	for i := range records {
		var openSatis float64
		// 1. Önce bugünün sabah açılış kaydını almayı dene
		err := stmtTodayOpen.QueryRowContext(ctx, records[i].Kod, startOfDay).Scan(&openSatis)
		if err != nil || openSatis == 0 {
			// 2. Yoksa dünün son kapanış kaydını almayı dene
			stmtPrevDayClose.QueryRowContext(ctx, records[i].Kod, startOfDay).Scan(&openSatis)
		}

		// 3. Hala yoksa en eski kaydı al
		if openSatis == 0 {
			r.db.DB().QueryRowContext(ctx, "SELECT satis FROM price_records WHERE kod = ? ORDER BY created_at ASC LIMIT 1", records[i].Kod).Scan(&openSatis)
		}

		if openSatis > 0 {
			fark := records[i].Satis - openSatis
			yuzde := (fark / openSatis) * 100.0

			records[i].Fark = math.Round(fark*10000) / 10000
			records[i].GunlukDegisimYuzde = math.Round(yuzde*100) / 100
		} else {
			records[i].Fark = 0
			records[i].GunlukDegisimYuzde = 0
		}
	}

	return records
}

// GetLatestAll - Tüm enstrümanların en son fiyat kayıtlarını getirir
// Her enstrüman kodu için sadece en son kaydı döndürür (subquery ile)
func (r *Repository) GetLatestAll(ctx context.Context) ([]models.PriceRecord, error) {
	// Her bir enstrüman (kod) için en son eklenen kaydı (max created_at) bulup birleştiren SQL sorgusu
	query := `
		SELECT p.id, p.kod, p.aciklama, p.alis, p.satis, p.kategori, p.guncelleme_zamani, p.created_at
		FROM price_records p
		INNER JOIN (
			SELECT kod, MAX(created_at) as max_created
			FROM price_records
			GROUP BY kod
		) latest ON p.kod = latest.kod AND p.created_at = latest.max_created
	`

	// Sorguyu çalıştır
	rows, err := r.db.DB().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("en son kayıtlar getirilemedi: %w", err)
	}
	defer rows.Close()

	var records []models.PriceRecord
	// Dönen tüm satırları sırayla oku
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	// Okuma sırasında gizli bir hata oluşup oluşmadığını kontrol et
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("satırları okuma hatası: %w", err)
	}

	return r.AttachDailyChanges(ctx, records), nil
}

// GetLatestByCategory - Belirli kategorideki en son fiyat kayıtlarını getirir
// category: "currency" veya "gold"
func (r *Repository) GetLatestByCategory(ctx context.Context, category string) ([]models.PriceRecord, error) {
	// Sadece seçilen kategori için her bir kodun en güncel kaydını çeken SQL sorgusu
	// WHERE filtresi subquery içine de eklenmiştir — performans için
	query := `
		SELECT p.id, p.kod, p.aciklama, p.alis, p.satis, p.kategori, p.guncelleme_zamani, p.created_at
		FROM price_records p
		INNER JOIN (
			SELECT kod, MAX(created_at) as max_created
			FROM price_records
			WHERE kategori = ?
			GROUP BY kod
		) latest ON p.kod = latest.kod AND p.created_at = latest.max_created
		WHERE p.kategori = ?
	`

	// Parametre olarak gelen kategori verisini sorguya enjekte ederek çalıştır
	// Kategori parametresi hem subquery hem de dış sorgu için gerekli
	rows, err := r.db.DB().QueryContext(ctx, query, category, category)
	if err != nil {
		return nil, fmt.Errorf("kategori bazlı en son kayıtlar getirilemedi: %w", err)
	}
	defer rows.Close()

	var records []models.PriceRecord
	// Tüm dönen sonuçları listeye ekle
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("satırları okuma hatası: %w", err)
	}

	return r.AttachDailyChanges(ctx, records), nil
}

// GetHistory - Belirli bir enstrümanın geçmiş fiyat kayıtlarını getirir
// kod: Enstrüman kodu (ör: "USD"), days: Kaç günlük geçmiş
// Grafik çizimi için kronolojik sıralama ile döndürür (eskiden yeniye)
func (r *Repository) GetHistory(ctx context.Context, kod string, days int) ([]models.PriceRecord, error) {
	// cutoff zamanını Go tarafında hesapla — SQLite'ın datetime('now') fonksiyonu yerine
	// Bu sayede timezone tutarsızlıklarını önlüyoruz
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")

	// Belirli koda ait verileri, cutoff zamanından itibaren kronolojik olarak getir
	query := `
		SELECT id, kod, aciklama, alis, satis, kategori, guncelleme_zamani, created_at
		FROM price_records
		WHERE kod = ? AND created_at >= ?
		ORDER BY created_at ASC
	`

	// Sorguyu ilgili kod ve cutoff zamanıyla çalıştır
	rows, err := r.db.DB().QueryContext(ctx, query, kod, cutoff)
	if err != nil {
		return nil, fmt.Errorf("geçmiş kayıtlar getirilemedi: %w", err)
	}
	defer rows.Close()

	var records []models.PriceRecord
	// Dönen sonuçları slice'a aktar
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("satırları okuma hatası: %w", err)
	}

	return records, nil
}

// GetHistoryAll - Tüm enstrümanların geçmiş fiyat kayıtlarını getirir
// days: Kaç günlük geçmiş isteniyor
func (r *Repository) GetHistoryAll(ctx context.Context, days int) ([]models.PriceRecord, error) {
	// cutoff zamanını Go tarafında hesapla
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")

	// Tüm kodlar için, cutoff zamanından itibaren olan bütün kayıtları getir
	query := `
		SELECT id, kod, aciklama, alis, satis, kategori, guncelleme_zamani, created_at
		FROM price_records
		WHERE created_at >= ?
		ORDER BY created_at ASC
	`

	// Sorguyu cutoff parametresiyle çalıştır
	rows, err := r.db.DB().QueryContext(ctx, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("tüm geçmiş kayıtlar getirilemedi: %w", err)
	}
	defer rows.Close()

	var records []models.PriceRecord
	// Dönen sonuçları slice'a aktar
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("satırları okuma hatası: %w", err)
	}

	return records, nil
}

// CleanupOldRecords - Belirtilen gün sayısından eski kayıtları siler
// retention: Tutulacak gün sayısı (ör: 30)
// Her gece scheduler tarafından çağrılır
func (r *Repository) CleanupOldRecords(ctx context.Context, retention int) (int64, error) {
	// cutoff zamanını Go tarafında hesapla
	cutoff := time.Now().UTC().AddDate(0, 0, -retention).Format("2006-01-02 15:04:05")

	// Cutoff'tan eski olan tüm verileri tablodan sil
	query := `DELETE FROM price_records WHERE created_at < ?`

	// Sadece silme yapacağımız için ExecContext kullanılır
	result, err := r.db.DB().ExecContext(ctx, query, cutoff)
	if err != nil {
		slog.Error("Eski kayıtlar silinemedi", "hata", err)
		return 0, fmt.Errorf("eski kayıtlar silinemedi: %w", err)
	}

	// Ne kadar verinin silindiğini kontrol et
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("etkilenen satır sayısı okunamadı: %w", err)
	}

	slog.Info("Eski kayıtlar başarıyla temizlendi", "silinen_kayit_sayisi", rowsAffected)
	return rowsAffected, nil
}

// SeedHistoricalDataIfEmpty - Veritabanında geçmiş 30 günlük veriler yoksa doldurur.
// Bu sayede 1 günlük, 7 günlük ve 30 günlük grafikler anlamlı şekilde ayrışır.
func (r *Repository) SeedHistoricalDataIfEmpty(ctx context.Context, currentRecords []models.PriceRecord) error {
	var count int
	// Son 2 güne ait kayıt var mı kontrol et
	err := r.db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM price_records WHERE created_at < datetime('now', '-1 days')").Scan(&count)
	if err != nil {
		return err
	}

	// Eğer 1 günden eski kayıtlar zaten varsa tohumlama yapma
	if count > 0 {
		return nil
	}

	slog.Info("Geçmiş 30 günlük örnek veriler dolduruluyor...", "mevcut_eski_kayit", count)

	tx, err := r.db.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO price_records (kod, aciklama, alis, satis, kategori, guncelleme_zamani, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer insertStmt.Close()

	now := time.Now().UTC()
	// Son 30 gün için her gün 2 örnek veri noktası oluştur (sabah & akşam)
	totalSeeded := 0
	for _, rec := range currentRecords {
		baseBuy := rec.Alis
		baseSell := rec.Satis

		// 30 gün geriden başla, bugüne kadar gel
		for day := 30; day >= 1; day-- {
			// Yumuşak, doğal sinüs dalgası fiyat hareketi (smooth trend)
			factor := 1.0 + math.Sin(float64(day)*0.35)*0.012

			pastTimeMorning := now.AddDate(0, 0, -day).Add(9 * time.Hour)
			pastTimeEvening := now.AddDate(0, 0, -day).Add(17 * time.Hour)

			guncelMorningStr := pastTimeMorning.Format("2006-01-02 15:04:05")
			guncelEveningStr := pastTimeEvening.Format("2006-01-02 15:04:05")

			// Sabah kaydı
			insertStmt.ExecContext(
				ctx,
				rec.Kod,
				rec.Aciklama,
				math.Round(baseBuy*factor*0.998*10000)/10000,
				math.Round(baseSell*factor*0.998*10000)/10000,
				rec.Kategori,
				guncelMorningStr,
				guncelMorningStr,
			)

			// Akşam kaydı
			insertStmt.ExecContext(
				ctx,
				rec.Kod,
				rec.Aciklama,
				math.Round(baseBuy*factor*1.002*10000)/10000,
				math.Round(baseSell*factor*1.002*10000)/10000,
				rec.Kategori,
				guncelEveningStr,
				guncelEveningStr,
			)

			totalSeeded += 2
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	slog.Info("Geçmiş 30 günlük veriler başarıyla tohumlandı!", "eklenen_gecmis_kayit", totalSeeded)
	return nil
}
