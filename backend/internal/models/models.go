package models

import (
	"strconv"
	"strings"
	"time"
)

// APIResponse - Altınkaynak API'sinden gelen ham JSON yanıt yapısı
type APIResponse struct {
	Alis              string `json:"Alis"`              // Alış fiyatı (Türk locale: "47,199")
	Satis             string `json:"Satis"`             // Satış fiyatı (Türk locale: "47,370")
	Kod               string `json:"Kod"`               // Enstrüman kodu (ör: "USD", "GA")
	Aciklama          string `json:"Aciklama"`          // Enstrüman açıklaması (ör: "Amerikan Doları")
	GuncellenmeZamani string `json:"GuncellenmeZamani"` // Son güncelleme zamanı (format: "27.07.2026 10:03:29")
}

// PriceRecord - Veritabanına kaydedilecek normalize edilmiş fiyat kaydı
type PriceRecord struct {
	ID                 int64     `json:"id"`                  // Benzersiz kayıt ID'si (auto-increment)
	Kod                string    `json:"kod"`                 // Enstrüman kodu
	Aciklama           string    `json:"aciklama"`            // Enstrüman açıklaması
	Alis               float64   `json:"alis"`                // Alış fiyatı (float64 olarak)
	Satis              float64   `json:"satis"`               // Satış fiyatı (float64 olarak)
	Kategori           string    `json:"kategori"`            // Kategori: "currency" veya "gold"
	GuncellenmeZamani  time.Time `json:"guncelleme_zamani"`   // API'deki güncelleme zamanı
	CreatedAt          time.Time `json:"created_at"`          // Veritabanına eklenme zamanı
	Fark               float64   `json:"fark"`                // Günlük fiyat farkı (TL)
	GunlukDegisimYuzde float64   `json:"gunluk_degisim_yuzde"` // Günlük % değişim (ör: +1.25 veya -0.50)
}

// WSMessage - WebSocket üzerinden gönderilen/alınan mesaj yapısı
type WSMessage struct {
	Type      string      `json:"type"`          // Mesaj tipi: "initial", "update", "history", "history_request"
	Data      interface{} `json:"data,omitempty"` // Mesaj verisi
	Kod       string      `json:"kod,omitempty"`  // Enstrüman kodu (history mesajları için)
	Timestamp string      `json:"timestamp"`      // Mesaj zaman damgası (ISO 8601)
}

// PriceData - Kategorilere ayrılmış fiyat verileri (initial ve update mesajlarında kullanılır)
type PriceData struct {
	Currency []PriceRecord `json:"currency"` // Döviz kurları listesi
	Gold     []PriceRecord `json:"gold"`     // Altın fiyatları listesi
}

// HistoryRequest - Client'tan gelen geçmiş veri talep mesajı
type HistoryRequest struct {
	Type string `json:"type"` // Mesaj tipi: "history_request"
	Kod  string `json:"kod"`  // İstenen enstrüman kodu
	Days int    `json:"days"` // Kaç günlük geçmiş isteniyor
}

// ParseTurkishPrice - Türkçe yerel ayarlı fiyat string'ini float64'e dönüştürür.
// Örnek: "6.209,70" -> 6209.70
// API'den gelen değerlerde bazen boşluk olabilir, bu yüzden önce trim edilir
func ParseTurkishPrice(s string) (float64, error) {
	// 0. Başındaki ve sonundaki boşlukları temizle
	s = strings.TrimSpace(s)
	// 1. Binlik ayırıcı olan noktaları kaldır
	s = strings.ReplaceAll(s, ".", "")
	// 2. Ondalık ayırıcı olan virgülü noktaya çevir
	s = strings.ReplaceAll(s, ",", ".")
	
	// 3. String'i float64 tipine çevir
	return strconv.ParseFloat(s, 64)
}

// ParseTurkishDateTime - "27.07.2026 10:03:29" formatındaki string'i time.Time yapısına dönüştürür.
// Turkey zaman dilimi (Europe/Istanbul) kullanılır.
func ParseTurkishDateTime(s string) (time.Time, error) {
	// Türkiye zaman dilimini (UTC+3) yükle
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		// Eğer lokasyon yüklenemezse geri dönüş hatasını bildir
		return time.Time{}, err
	}
	
	// Formatlanmış zamanı ilgili lokasyonda çözümle
	return time.ParseInLocation("02.01.2006 15:04:05", s, loc)
}
