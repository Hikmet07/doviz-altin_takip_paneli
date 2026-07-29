import { useEffect, useRef, useState } from "react";
import { normalizePriceRecord } from "../services/api";

function getSocketUrl() {
  let url =
    import.meta.env.VITE_SOCKET_URL ||
    "wss://doviz-altin-takip-paneli.onrender.com/ws";
  url = url.trim();

  if (url.startsWith("wshttp://")) {
    url = url.replace(/^wshttp:\/\//, "ws://");
  } else if (url.startsWith("wshttps://")) {
    url = url.replace(/^wshttps:\/\//, "wss://");
  } else if (url.startsWith("https://")) {
    url = url.replace(/^https:\/\//, "wss://");
  } else if (url.startsWith("http://")) {
    url = url.replace(/^http:\/\//, "ws://");
  } else if (!url.startsWith("ws://") && !url.startsWith("wss://")) {
    url = "wss://" + url;
  }

  url = url.replace(/([^:]\/)\/+/g, "$1");

  if (!url.endsWith("/ws")) {
    url = url.replace(/\/+$/, "") + "/ws";
  }

  return url;
}

function useMarketSocket() {
  const socketRef = useRef(null);
  const reconnectTimerRef = useRef(null);
  const previousPricesRef = useRef(new Map());

  const [liveData, setLiveData] = useState([]);
  const [socketStatus, setSocketStatus] = useState("connecting");
  const [statusText, setStatusText] = useState("Bağlanıyor...");
  const [lastUpdate, setLastUpdate] = useState(null);

  useEffect(() => {
    function connect() {
      setSocketStatus("connecting");
      setStatusText("Bağlanıyor...");

      try {
        const socketUrl = getSocketUrl();
        const socket = new WebSocket(socketUrl);
        socketRef.current = socket;

        socket.onopen = () => {
          setSocketStatus("connected");
          setStatusText("Canlı Bağlantı Aktif");
        };

        socket.onmessage = (event) => {
          try {
            const message = JSON.parse(event.data);

            if (message.type === "update") {
              const currencyData = Array.isArray(message.data?.currency)
                ? message.data.currency
                : [];

              const goldData = Array.isArray(message.data?.gold)
                ? message.data.gold
                : [];

              const rawList = [...currencyData, ...goldData];
              const prevPrices = previousPricesRef.current;

              const mergedData = rawList.map((record) => {
                const normalized = normalizePriceRecord(record);
                const prev = prevPrices.get(normalized.symbol);

                if (prev) {
                  const instantDiff = normalized.sellPrice - prev.sellPrice;
                  const instantPercent =
                    prev.sellPrice > 0
                      ? Number(((instantDiff / prev.sellPrice) * 100).toFixed(2))
                      : 0;

                  if (instantDiff > 0) {
                    normalized.instantTrend = "up";
                    normalized.instantDiff = instantDiff;
                    normalized.instantPercent = instantPercent;
                  } else if (instantDiff < 0) {
                    normalized.instantTrend = "down";
                    normalized.instantDiff = instantDiff;
                    normalized.instantPercent = instantPercent;
                  } else {
                    normalized.instantTrend = prev.instantTrend || "same";
                    normalized.instantDiff = prev.instantDiff || 0;
                    normalized.instantPercent = prev.instantPercent || 0;
                  }
                } else {
                  normalized.instantTrend = "same";
                  normalized.instantDiff = 0;
                  normalized.instantPercent = 0;
                }

                prevPrices.set(normalized.symbol, {
                  sellPrice: normalized.sellPrice,
                  instantTrend: normalized.instantTrend,
                  instantDiff: normalized.instantDiff,
                  instantPercent: normalized.instantPercent,
                });

                return normalized;
              });

              setLiveData(mergedData);
              setLastUpdate(new Date().toLocaleTimeString("tr-TR"));
            }
          } catch (error) {
            console.error("WebSocket mesajı okunamadı:", error);
          }
        };

        socket.onerror = (err) => {
          console.warn("WebSocket bağlantı uyarısı:", err);
          setSocketStatus("connecting");
          setStatusText("Yeniden Bağlanılıyor...");
        };

        socket.onclose = () => {
          setSocketStatus("connecting");
          setStatusText("Yeniden Bağlanılıyor...");

          reconnectTimerRef.current = setTimeout(() => {
            connect();
          }, 3000);
        };
      } catch (err) {
        console.error("WebSocket kurulum hatası:", err);
        setSocketStatus("connecting");
        setStatusText("Yeniden Bağlanılıyor...");
      }
    }

    connect();

    return () => {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
      }

      if (socketRef.current) {
        socketRef.current.close();
      }
    };
  }, []);

  return { liveData, socketStatus, statusText, lastUpdate };
}

export default useMarketSocket;