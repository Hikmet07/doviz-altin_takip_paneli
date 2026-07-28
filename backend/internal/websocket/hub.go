package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/altinkaynak-bot/backend/internal/models"
	ws "github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second    // Yazma işlemi için maksimum bekleme süresi
	pongWait       = 60 * time.Second    // Pong mesajı için maksimum bekleme süresi
	pingPeriod     = (pongWait * 9) / 10 // Ping gönderme aralığı (pongWait'in %90'ı)
	maxMessageSize = 4096                // Client'tan alınabilecek maksimum mesaj boyutu (history_request gibi mesajlar için yeterli)
)

// upgrader - HTTP bağlantısını WebSocket'e yükselten yapılandırma
var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Geliştirme ortamı için tüm origin'lere izin veriyoruz
	// Prodüksiyon ortamında bu fonksiyon kısıtlanmalıdır
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client - Tek bir WebSocket bağlantısını temsil eder
type Client struct {
	hub  *Hub        // Bağlı olduğu merkezi hub
	conn *ws.Conn    // Aktif WebSocket bağlantısı
	send chan []byte  // Client'a gönderilecek mesaj kuyruğu (buffered channel)
}

// Hub - Tüm aktif WebSocket bağlantılarını yöneten merkezi yapı
// Register/Unregister/Broadcast işlemlerini tek bir goroutine üzerinden yönetir (thread-safe)
type Hub struct {
	clients    map[*Client]bool // Aktif client'ları tutan harita
	broadcast  chan []byte      // Tüm client'lara yayınlanacak mesajlar
	register   chan *Client     // Yeni client kayıt kanalı
	unregister chan *Client     // Client çıkış kanalı
	mu         sync.RWMutex    // onMessage callback'i için mutex
	onMessage  func([]byte)    // Client'tan mesaj geldiğinde çağrılacak callback
}

// NewHub - Yeni bir Hub örneği oluşturur ve döndürür
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run - Hub'ı başlatır ve gelen register/unregister/broadcast işlemlerini dinler
// Bu fonksiyon tek bir goroutine'de çalışır, dolayısıyla clients map'ine erişim
// zaten seri halde yapılır — ek mutex gerekmez (channel-based synchronization)
func (h *Hub) Run() {
	for {
		select {
		// Yeni bir client bağlandığında
		case client := <-h.register:
			h.clients[client] = true
			slog.Info("Yeni WebSocket client'ı bağlandı", "client_count", len(h.clients))

		// Bir client ayrıldığında
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send) // Mesaj kanalını kapat
			}
			slog.Info("WebSocket client'ı ayrıldı", "client_count", len(h.clients))

		// Tüm client'lara mesaj yayınlanacağı zaman
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message: // Mesajı gönder
				default:
					// Gönderilemiyorsa (ör. buffer dolu, yavaş client) bağlantıyı temizle
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// Broadcast - Tüm client'lara raw byte mesajı yayınlar
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// BroadcastMessage - WSMessage tipindeki veriyi JSON formatına çevirip yayınlar
func (h *Hub) BroadcastMessage(msg models.WSMessage) error {
	// WSMessage nesnesini JSON byte dizisine dönüştür
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("Mesaj JSON'a çevrilemedi", "error", err)
		return err
	}
	h.Broadcast(data)
	return nil
}

// SetOnMessage - Client'tan bir mesaj geldiğinde çalıştırılacak fonksiyonu belirler
// Bu callback, readPump goroutine'lerinden çağrılacağı için mutex ile korunur
func (h *Hub) SetOnMessage(fn func([]byte)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onMessage = fn
}

// ClientCount - Aktif olarak bağlı olan client sayısını döndürür
// NOT: Bu değer yaklaşık bir değerdir çünkü Run() goroutine'i dışında çağrılır
func (h *Hub) ClientCount() int {
	// broadcast kanalı üzerinden senkronize olmadığımız için
	// anlık bir snapshot döndürüyoruz
	return len(h.clients)
}

// HandleWebSocket - Gelen HTTP isteğini WebSocket bağlantısına yükseltir
// ve okuma/yazma pump goroutine'lerini başlatır
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// HTTP bağlantısını WebSocket protokolüne yükselt
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade hatası", "error", err)
		return
	}

	// Yeni client nesnesi oluştur (256 mesajlık buffer ile)
	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	// Client'ı Hub'a kaydet
	client.hub.register <- client

	// Okuma ve yazma işlemlerini ayrı goroutine'lerde başlat
	// writePump önce başlatılmalı ki readPump kapandığında
	// unregister mesajı send kanalına düzgünce iletilsin
	go client.writePump()
	go client.readPump()
}

// readPump - Client'tan gelen mesajları okur ve işler
// Her client için ayrı bir goroutine'de çalışır
// Bağlantı koptuğunda veya hata oluştuğunda goroutine sonlanır
func (c *Client) readPump() {
	defer func() {
		// Bağlantı koptuğunda veya hata olduğunda temizlik yap
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// Client'tan alınabilecek maksimum mesaj boyutunu ayarla
	c.conn.SetReadLimit(maxMessageSize)
	// İlk okuma için deadline ayarla
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	// Pong mesajı geldiğinde okuma deadline'ını uzat
	// Bu sayede bağlantının canlı olduğunu anlıyoruz
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		// Client'tan mesaj oku (blocking)
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			// Beklenmeyen kapanma hatalarını logla
			if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				slog.Error("Beklenmeyen WebSocket kapanması", "error", err)
			}
			break
		}

		// Eğer mesaj callback'i ayarlanmışsa, gelen mesajı oraya ilet
		// Mutex ile korunuyor çünkü onMessage farklı bir goroutine'den ayarlanabilir
		c.hub.mu.RLock()
		onMsg := c.hub.onMessage
		c.hub.mu.RUnlock()

		if onMsg != nil {
			onMsg(message)
		}
	}
}

// writePump - Sunucudan client'a mesajları gönderir
// Her client için ayrı bir goroutine'de çalışır
// Periyodik olarak ping mesajı göndererek bağlantı sağlığını kontrol eder
func (c *Client) writePump() {
	// Periyodik ping göndermek için ticker oluştur
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		// Gönderilecek mesaj geldiğinde
		case message, ok := <-c.send:
			// Yazma işlemi için deadline ayarla
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// send kanalı kapatılmış (Hub tarafından) — bağlantıyı kapat
				c.conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}

			// Her mesajı ayrı bir WebSocket frame olarak gönder
			// NOT: Birden fazla JSON mesajını tek frame'de birleştirmiyoruz
			// çünkü frontend her frame'i ayrı bir JSON mesajı olarak parse eder
			err := c.conn.WriteMessage(ws.TextMessage, message)
			if err != nil {
				slog.Error("WebSocket mesaj gönderme hatası", "error", err)
				return
			}

		// Periyodik olarak ping mesajı gönder (bağlantı canlılık kontrolü)
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(ws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
