import { useEffect, useRef, useState } from "react";
import { normalizePriceRecord } from "../services/api";

const SOCKET_URL =
  import.meta.env.VITE_SOCKET_URL || "wss://doviz-altin-takip-paneli.onrender.com/ws";

function useMarketSocket() {
  const socketRef = useRef(null);
  const reconnectTimerRef = useRef(null);
  const previousPricesRef = useRef(new Map());

  const [liveData, setLiveData] = useState([]);
  const [socketStatus, setSocketStatus] = useState("connecting"); // "connected", "connecting", "error", "disconnected"
  const [statusText, setStatusText] = useState("Bağlanıyor...");
  const [lastUpdate, setLastUpdate] = useState(null);

  useEffect(() => {
    function connect() {
      setSocketStatus("connecting");
      setStatusText("Bağlanıyor...");

      const socket = new WebSocket(SOCKET_URL);
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
                const instantPercent = prev.sellPrice > 0 ? Number(((instantDiff / prev.sellPrice) * 100).toFixed(2)) : 0;

                if (instantDiff > 0) {
                  normalized.instantTrend = "up";
                  normalized.instantDiff = instantDiff;
                  normalized.instantPercent = instantPercent;
                } else if (instantDiff < 0) {
                  normalized.instantTrend = "down";
                  normalized.instantDiff = instantDiff;
                  normalized.instantPercent = instantPercent;
                } else {
                  // Son gerçekleşen canlı tik hareketini koru (Borsa İstanbul ticker davranışı)
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

      socket.onerror = () => {
        setSocketStatus("error");
        setStatusText("Bağlantı Hatası");
      };

      socket.onclose = () => {
        setSocketStatus("disconnected");
        setStatusText("Bağlantı kapandı, 3s sonra tekrar deneniyor...");

        reconnectTimerRef.current = setTimeout(() => {
          connect();
        }, 3000);
      };
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