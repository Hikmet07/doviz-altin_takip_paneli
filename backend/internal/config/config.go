package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config - Uygulama genelinde kullanılacak ayarların tutulduğu yapı
type Config struct {
	ServerPort           string        // HTTP sunucu portu
	DBPath               string        // SQLite veritabanı dosya yolu
	FetchInterval        time.Duration // API'den veri çekme aralığı
	CleanupRetentionDays int           // Veritabanında tutulacak maksimum gün sayısı
	CORSOrigins          []string      // İzin verilen CORS origin'leri
}

// Load - Çevresel değişkenlerden (ve varsa .env dosyasından) konfigürasyonu yükler
func Load() *Config {
	// İsteğe bağlı olarak .env dosyasını yüklemeyi dene
	err := godotenv.Load()
	if err != nil {
		// .env dosyası bulunamazsa sadece bilgi ver, hata fırlatma
		log.Println("Bilgi: .env dosyası bulunamadı, mevcut çevre değişkenleri kullanılacak.")
	}

	cfg := &Config{
		ServerPort:           getEnv("SERVER_PORT", "8080"),
		DBPath:               getEnv("DB_PATH", "./data/market.db"),
		FetchInterval:        getEnvAsDuration("FETCH_INTERVAL", 30*time.Second),
		CleanupRetentionDays: getEnvAsInt("CLEANUP_RETENTION_DAYS", 30),
		CORSOrigins:          strings.Split(getEnv("CORS_ORIGINS", "*"), ","),
	}

	return cfg
}

// getEnv - Belirtilen çevre değişkenini okur, yoksa varsayılan değeri döndürür
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// getEnvAsInt - Belirtilen çevre değişkenini int olarak okur, dönüştürülemezse veya yoksa varsayılanı döndürür
func getEnvAsInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return fallback
}

// getEnvAsDuration - Belirtilen çevre değişkenini time.Duration olarak okur
func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return fallback
}
