package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Database - SQLite veritabanı bağlantısını yöneten yapı
// Bu yapı, SQLite veritabanına olan aktif bağlantıyı güvenli bir şekilde tutar.
type Database struct {
	// db - Alt seviye sql.DB nesnesi, sorguların çalıştırılması için kullanılır
	db *sql.DB
}

// NewDatabase - Yeni bir SQLite veritabanı bağlantısı oluşturur
// dbPath: Veritabanı dosya yolu (ör: "./data/market.db")
// Dosya yoksa oluşturur, WAL mode aktifleştirir, migration çalıştırır
func NewDatabase(dbPath string) (*Database, error) {
	// Veritabanı dosyasının bulunacağı dizini al
	dir := filepath.Dir(dbPath)

	// Dizin yoksa, gerekli tüm üst dizinlerle birlikte oluştur (os.MkdirAll kullanarak)
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error("Veritabanı dizini oluşturulamadı", "hata", err, "dizin", dir)
		return nil, fmt.Errorf("veritabanı dizini oluşturulamadı: %w", err)
	}

	// SQLite veritabanına bağlan
	// modernc.org/sqlite sürücüsünü (driver name: "sqlite") kullanıyoruz
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		slog.Error("Veritabanına bağlanılamadı", "hata", err)
		return nil, fmt.Errorf("veritabanına bağlanılamadı: %w", err)
	}

	// Bağlantının çalışıp çalışmadığını ping atarak kontrol et
	if err := db.Ping(); err != nil {
		db.Close()
		slog.Error("Veritabanı ping başarısız", "hata", err)
		return nil, fmt.Errorf("veritabanı ping başarısız: %w", err)
	}

	// Performansı ve eşzamanlılığı artırmak için WAL (Write-Ahead Logging) modunu aktifleştir
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		slog.Warn("WAL modu aktifleştirilemedi", "hata", err)
	}

	// Aynı anda birden fazla işlemin veritabanını kilitlemesini önlemek için zaman aşımı (5000ms) süresini ayarla
	if _, err := db.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		slog.Warn("Busy timeout ayarlanamadı", "hata", err)
	}

	// Database yapısını oluştur ve nesneye ata
	database := &Database{
		db: db,
	}

	// Veritabanı tablolarını ve indexleri oluştur
	if err := database.migrate(); err != nil {
		db.Close()
		slog.Error("Veritabanı migration başarısız", "hata", err)
		return nil, fmt.Errorf("migration başarısız: %w", err)
	}

	slog.Info("Veritabanı bağlantısı başarıyla oluşturuldu", "yol", dbPath)
	return database, nil
}

// migrate - Veritabanı tablolarını ve indexleri oluşturur
// Uygulamanın ihtiyacı olan tablo şemalarını ve performansı artıracak indeksleri ekler
func (d *Database) migrate() error {
	// Tablo ve indexleri oluşturmak için gerekli SQL sorgusu
	query := `
	CREATE TABLE IF NOT EXISTS price_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		kod TEXT NOT NULL,
		aciklama TEXT NOT NULL,
		alis REAL NOT NULL,
		satis REAL NOT NULL,
		kategori TEXT NOT NULL,
		guncelleme_zamani DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_kod ON price_records(kod);
	CREATE INDEX IF NOT EXISTS idx_kategori ON price_records(kategori);
	CREATE INDEX IF NOT EXISTS idx_created_at ON price_records(created_at);
	CREATE INDEX IF NOT EXISTS idx_kod_created ON price_records(kod, created_at);
	`

	// SQL sorgusunu veritabanı üzerinde çalıştır
	_, err := d.db.Exec(query)
	if err != nil {
		return fmt.Errorf("tablolar oluşturulamadı: %w", err)
	}

	slog.Info("Veritabanı migration işlemi başarıyla tamamlandı")
	return nil
}

// Close - Veritabanı bağlantısını kapatır
// Uygulama kapanırken açık olan bağlantıları serbest bırakmak için çağrılmalıdır.
func (d *Database) Close() error {
	slog.Info("Veritabanı bağlantısı kapatılıyor")
	return d.db.Close()
}

// DB - Underlying sql.DB nesnesine erişim sağlar
// Gerekli olduğunda transaction yönetimi veya alt seviye sorgular için bu nesneye ulaşılır.
func (d *Database) DB() *sql.DB {
	return d.db
}
