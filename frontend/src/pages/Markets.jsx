import { useEffect, useState } from "react";
import { getMarkets } from "../services/api";
import useMarketSocket from "../hooks/useMarketSocket";

import MarketTable from "../components/MarketTable";
import Loading from "../components/Loading";
import ErrorMessage from "../components/ErrorMessage";

function Markets() {
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
          "Piyasa verileri alınamadı. Backend çalışıyor mu kontrol edin."
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

  if (loading) {
    return <Loading text="Piyasa verileri yükleniyor..." />;
  }

  if (errorMessage) {
    return <ErrorMessage message={errorMessage} />;
  }

  const currencyCount = markets.filter((m) => m.type === "currency").length;
  const goldCount = markets.filter((m) => m.type === "gold").length;

  return (
    <div className="space-y-6">
      {/* Header Banner */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between rounded-2xl bg-white p-6 border border-slate-200 shadow-xs">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 tracking-tight">Tüm Piyasalar</h1>
          <p className="mt-1 text-sm text-slate-500">
            Canlı Altın Fiyatları ve Döviz Kurları Listesi
          </p>
        </div>

        <div className="flex items-center gap-2 text-xs font-semibold">
          <span className="rounded-xl bg-blue-50 border border-blue-200/60 px-3 py-1.5 text-blue-700">
            {currencyCount} Döviz Kuru
          </span>
          <span className="rounded-xl bg-amber-50 border border-amber-200/60 px-3 py-1.5 text-amber-700">
            {goldCount} Altın Çeşidi
          </span>
        </div>
      </div>

      <MarketTable markets={markets} />
    </div>
  );
}

export default Markets;
