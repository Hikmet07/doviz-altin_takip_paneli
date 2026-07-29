import { useEffect, useState } from "react";
import { getHistory, getSymbols } from "../services/api";

import PriceChart from "../components/PriceChart";
import Loading from "../components/Loading";
import ErrorMessage from "../components/ErrorMessage";

function History() {
  const [symbols, setSymbols] = useState([]);
  const [selectedSymbol, setSelectedSymbol] = useState("");
  const [selectedRange, setSelectedRange] = useState("30d");
  const [historyData, setHistoryData] = useState([]);
  const [symbolsLoading, setSymbolsLoading] = useState(true);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    const fetchSymbols = async () => {
      try {
        const data = await getSymbols();
        const symbolList = Array.isArray(data) ? data : [];

        setSymbols(symbolList);

        if (symbolList.length > 0) {
          const defaultSym = symbolList.find((s) => s.symbol === "USD") || symbolList[0];
          setSelectedSymbol(defaultSym.symbol);
        }
      } catch (error) {
        console.error(error);
        setErrorMessage(
          "Sembol listesi alınamadı. Backend çalışıyor mu kontrol edin."
        );
      } finally {
        setSymbolsLoading(false);
      }
    };

    fetchSymbols();
  }, []);

  useEffect(() => {
    if (!selectedSymbol) return;

    const fetchHistory = async () => {
      try {
        setHistoryLoading(true);
        setErrorMessage("");

        const data = await getHistory(selectedSymbol, selectedRange);
        setHistoryData(Array.isArray(data) ? data : []);
      } catch (error) {
        console.error(error);
        setErrorMessage("Geçmiş veriler alınamadı.");
      } finally {
        setHistoryLoading(false);
      }
    };

    fetchHistory();
  }, [selectedSymbol, selectedRange]);

  if (symbolsLoading) {
    return <Loading text="Sembol listesi yükleniyor..." />;
  }

  if (errorMessage && symbols.length === 0) {
    return <ErrorMessage message={errorMessage} />;
  }

  const activeSymbolObj = symbols.find((s) => s.symbol === selectedSymbol);
  const latestRecord = historyData.length > 0 ? historyData[historyData.length - 1] : null;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between rounded-2xl bg-white p-6 border border-slate-200 shadow-xs">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 tracking-tight">
            Piyasa Grafikleri
          </h1>
          <p className="mt-1 text-sm text-slate-500">
            Altın ve Döviz kurlarının zamana göre değişim analizleri
          </p>
        </div>

        <div className="inline-flex rounded-xl bg-slate-100 p-1 text-xs font-semibold text-slate-600">
          {[
            { id: "1d", label: "Son 1 Gün" },
            { id: "7d", label: "Son 7 Gün" },
            { id: "30d", label: "Son 30 Gün" },
          ].map((range) => (
            <button
              key={range.id}
              onClick={() => setSelectedRange(range.id)}
              className={`rounded-lg px-3.5 py-1.5 transition-all ${
                selectedRange === range.id
                  ? "bg-white text-slate-900 shadow-xs"
                  : "hover:text-slate-900"
              }`}
            >
              {range.label}
            </button>
          ))}
        </div>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-xs">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3 md:items-end">
          <div className="md:col-span-2">
            <label className="mb-2 block text-xs font-bold uppercase tracking-wider text-slate-500">
              Enstrüman Seç
            </label>

            <select
              value={selectedSymbol}
              onChange={(e) => setSelectedSymbol(e.target.value)}
              className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-2.5 text-sm font-semibold text-slate-900 outline-none transition-all focus:border-indigo-500 focus:bg-white focus:ring-2 focus:ring-indigo-100"
            >
              {symbols.map((sym) => (
                <option key={sym.symbol} value={sym.symbol}>
                  {sym.symbol} - {sym.name} ({sym.typeLabel})
                </option>
              ))}
            </select>
          </div>

          {latestRecord && (
            <div className="rounded-xl bg-slate-50 border border-slate-100 p-3 text-right">
              <span className="text-xs text-slate-400 font-medium">Son Güncellenen Satış Fiyatı</span>
              <p className="text-lg font-bold text-indigo-600 tabular-nums">
                {latestRecord.sellPrice.toLocaleString("tr-TR", {
                  minimumFractionDigits: 2,
                  maximumFractionDigits: 4,
                })}{" "}
                ₺
              </p>
            </div>
          )}
        </div>
      </div>

      {errorMessage && <ErrorMessage message={errorMessage} />}

      <div>
        {historyLoading ? (
          <Loading text="Grafik verisi yükleniyor..." />
        ) : (
          <PriceChart
            data={historyData}
            range={selectedRange}
            title={`${activeSymbolObj ? `${activeSymbolObj.name} (${activeSymbolObj.symbol})` : selectedSymbol} Fiyat Grafiği`}
          />
        )}
      </div>
    </div>
  );
}

export default History;