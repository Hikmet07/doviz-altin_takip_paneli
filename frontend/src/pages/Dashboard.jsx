import { useEffect, useState } from "react";
import { getMarkets } from "../services/api";
import useMarketSocket from "../hooks/useMarketSocket";

import MarketCard from "../components/MarketCard";
import MarketTable from "../components/MarketTable";
import Loading from "../components/Loading";
import ErrorMessage from "../components/ErrorMessage";

function Dashboard() {
  const [markets, setMarkets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  const { liveData, socketStatus, statusText, lastUpdate } = useMarketSocket();

  useEffect(() => {
    const fetchMarkets = async () => {
      try {
        const data = await getMarkets();
        setMarkets(Array.isArray(data) ? data : []);
      } catch (error) {
        console.error(error);
        setErrorMessage(
          "Piyasa verileri alınamadı. Backend sunucusunun (http://localhost:8080) çalıştığından emin olun."
        );
      } finally {
        setLoading(false);
      }
    };

    fetchMarkets();
  }, []);

  useEffect(() => {
    if (liveData.length > 0) {
      setMarkets(liveData);
    }
  }, [liveData]);

  const benchmarkSymbols = ["USD", "EUR", "GA", "PC"];
  const featuredMarkets = benchmarkSymbols
    .map((sym) => markets.find((m) => m.symbol === sym))
    .filter(Boolean);

  const fallbackMarkets =
    featuredMarkets.length >= 4 ? featuredMarkets : markets.slice(0, 4);

  if (loading) {
    return <Loading text="Piyasa verileri yükleniyor..." />;
  }

  if (errorMessage) {
    return <ErrorMessage message={errorMessage} />;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between rounded-2xl bg-white p-6 border border-slate-200 shadow-xs">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 tracking-tight">
            Piyasa Dashboard
          </h1>
          <p className="mt-1 text-sm text-slate-500">
            Canlı Altın Fiyatları ve Döviz Kurları Takip Ekranı
          </p>
        </div>

        <div className="flex items-center gap-3">
          <div className="inline-flex items-center gap-2 rounded-xl bg-slate-50 border border-slate-200 px-3.5 py-2 text-xs font-semibold">
            <span
              className={`h-2.5 w-2.5 rounded-full ${
                socketStatus === "connected"
                  ? "bg-emerald-500 animate-pulse"
                  : socketStatus === "connecting"
                  ? "bg-amber-500 animate-ping"
                  : "bg-rose-500"
              }`}
            ></span>
            <span className="text-slate-700">{statusText}</span>
          </div>

          {lastUpdate && (
            <div className="hidden sm:block text-xs font-medium text-slate-500 bg-slate-50 border border-slate-200 px-3 py-2 rounded-xl">
              Son Güncelleme: <span className="font-bold text-slate-800">{lastUpdate}</span>
            </div>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {fallbackMarkets.map((market) => (
          <MarketCard
            key={market.symbol}
            title={market.name}
            symbol={market.symbol}
            buyPrice={market.buyPrice}
            sellPrice={market.sellPrice}
            changePercent={market.changePercent}
            trend={market.trend}
            typeLabel={market.typeLabel}
          />
        ))}
      </div>

      <MarketTable markets={markets} />
    </div>
  );
}

export default Dashboard;