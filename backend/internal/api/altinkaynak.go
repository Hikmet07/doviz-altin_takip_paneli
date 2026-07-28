package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/altinkaynak-bot/backend/internal/models"
)

// Client - Altınkaynak API istemcisi
type Client struct {
	httpClient *http.Client
}

// NewClient - Yeni bir Altınkaynak API istemcisi oluşturur
func NewClient() *Client {
	// HTTP istemcisi için genel bir zaman aşımı (15 saniye) belirle
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// fetchData - Belirtilen URL'den veri çeker, tekrar deneme (retry) mekanizmasını içerir
func (c *Client) fetchData(ctx context.Context, url string, category string) ([]models.PriceRecord, error) {
	var sonHata error
	
	// İstek başarısız olursa maksimum 3 defa tekrar dene
	for deneme := 1; deneme <= 3; deneme++ {
		// Context'in iptal edilip edilmediğini kontrol et
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context iptal edildi: %w", err)
		}

		slog.Info("API isteği yapılıyor", "url", url, "kategori", category, "deneme", deneme)
		
		// Yeni HTTP isteği oluştur
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("istek oluşturulamadı: %w", err)
		}
		
		// İsteği gerçekleştir
		resp, err := c.httpClient.Do(req)
		if err != nil {
			sonHata = err
			slog.Warn("API isteği başarısız oldu", "hata", err, "kategori", category)
			// Eğer son deneme değilse 2 saniye bekle
			if deneme < 3 {
				time.Sleep(2 * time.Second)
			}
			continue
		}
		
		// Başarılı bir yanıt alınmadıysa
		if resp.StatusCode != http.StatusOK {
			sonHata = fmt.Errorf("beklenmeyen HTTP durumu: %d", resp.StatusCode)
			slog.Warn("API'den hatalı durum kodu döndü", "kod", resp.StatusCode, "kategori", category)
			resp.Body.Close()
			if deneme < 3 {
				time.Sleep(2 * time.Second)
			}
			continue
		}
		
		// Yanıt gövdesini oku
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("yanıt okunamadı: %w", err)
		}
		
		// JSON yanıtı çözümle
		var apiResponses []models.APIResponse
		if err := json.Unmarshal(body, &apiResponses); err != nil {
			return nil, fmt.Errorf("JSON parse hatası: %w", err)
		}
		
		// Gelen ham API yanıtlarını normalize edilmiş yapıya (PriceRecord) dönüştür
		var records []models.PriceRecord
		for _, apiRes := range apiResponses {
			// Alış fiyatını çevir
			alis, err := models.ParseTurkishPrice(apiRes.Alis)
			if err != nil {
				slog.Error("Alış fiyatı dönüştürülemedi", "kod", apiRes.Kod, "değer", apiRes.Alis)
				continue
			}
			
			// Satış fiyatını çevir
			satis, err := models.ParseTurkishPrice(apiRes.Satis)
			if err != nil {
				slog.Error("Satış fiyatı dönüştürülemedi", "kod", apiRes.Kod, "değer", apiRes.Satis)
				continue
			}
			
			// Güncellenme zamanını çevir
			guncellemeZamani, err := models.ParseTurkishDateTime(apiRes.GuncellenmeZamani)
			if err != nil {
				slog.Error("Zaman dönüştürülemedi", "kod", apiRes.Kod, "değer", apiRes.GuncellenmeZamani)
				continue
			}
			
			// Kaydı listeye ekle
			records = append(records, models.PriceRecord{
				Kod:               apiRes.Kod,
				Aciklama:          apiRes.Aciklama,
				Alis:              alis,
				Satis:             satis,
				Kategori:          category,
				GuncellenmeZamani: guncellemeZamani,
				CreatedAt:         time.Now(),
			})
		}
		
		// Başarılı olursa döngüden çık ve kayıtları dön
		return records, nil
	}
	
	// Tüm denemeler başarısız olduysa son hatayı dön
	return nil, fmt.Errorf("3 deneme sonrasında api isteği başarısız: %w", sonHata)
}

// FetchCurrency - Döviz kurlarını çeker ve PriceRecord listesi döndürür
func (c *Client) FetchCurrency(ctx context.Context) ([]models.PriceRecord, error) {
	// Döviz kurları için belirlenmiş statik URL
	url := "https://static.altinkaynak.com/public/Currency"
	return c.fetchData(ctx, url, "currency")
}

// FetchGold - Altın fiyatlarını çeker ve PriceRecord listesi döndürür  
func (c *Client) FetchGold(ctx context.Context) ([]models.PriceRecord, error) {
	// Altın fiyatları için belirlenmiş statik URL
	url := "https://static.altinkaynak.com/public/Gold"
	return c.fetchData(ctx, url, "gold")
}

// FetchAll - Tüm verileri paralel olarak çeker (currency + gold)
func (c *Client) FetchAll(ctx context.Context) ([]models.PriceRecord, error) {
	var (
		wg         sync.WaitGroup
		allRecords []models.PriceRecord
		mu         sync.Mutex // allRecords listesine eşzamanlı erişimi korumak için
		finalErr   error
		errMu      sync.Mutex // Hata değişkenine eşzamanlı erişimi korumak için
	)
	
	// İki farklı (Döviz ve Altın) istek yapılacağı için WaitGroup'a 2 ekliyoruz
	wg.Add(2)
	
	// Döviz verilerini çekmek için goroutine
	go func() {
		defer wg.Done()
		
		currencyRecords, err := c.FetchCurrency(ctx)
		if err != nil {
			errMu.Lock()
			finalErr = err
			errMu.Unlock()
			slog.Error("Döviz verileri çekilemedi", "hata", err)
			return
		}
		
		// Başarılı ise genel listeye ekle (Mutex ile güvenli erişim)
		mu.Lock()
		allRecords = append(allRecords, currencyRecords...)
		mu.Unlock()
	}()
	
	// Altın verilerini çekmek için goroutine
	go func() {
		defer wg.Done()
		
		goldRecords, err := c.FetchGold(ctx)
		if err != nil {
			errMu.Lock()
			finalErr = err
			errMu.Unlock()
			slog.Error("Altın verileri çekilemedi", "hata", err)
			return
		}
		
		// Başarılı ise genel listeye ekle (Mutex ile güvenli erişim)
		mu.Lock()
		allRecords = append(allRecords, goldRecords...)
		mu.Unlock()
	}()
	
	// Tüm goroutine'lerin tamamlanmasını bekle
	wg.Wait()
	
	// Eğer herhangi bir işlem hata verdiyse hatayı döndür
	if finalErr != nil {
		return nil, fmt.Errorf("veri çekme işlemlerinden en az biri başarısız oldu: %w", finalErr)
	}
	
	// Başarılı şekilde çekilen ve birleştirilen tüm verileri döndür
	return allRecords, nil
}
