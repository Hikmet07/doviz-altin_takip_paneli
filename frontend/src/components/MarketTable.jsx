import { useState } from "react";

function MarketTable({ markets }) {
  const [searchQuery, setSearchQuery] = useState("");
  const [activeTab, setActiveTab] = useState("all");

  const filteredMarkets = markets.filter((market) => {
    const matchesTab =
      activeTab === "all" || market.type === activeTab;

    const matchesSearch =
      market.symbol.toLowerCase().includes(searchQuery.toLowerCase()) ||
      market.name.toLowerCase().includes(searchQuery.toLowerCase());

    return matchesTab && matchesSearch;
  });

  return (
    <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div className="border-b border-slate-100 p-5">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 className="text-xl font-bold text-slate-900 flex items-center gap-2">
              <span className="inline-block h-3 w-3 rounded-full bg-emerald-500 animate-pulse"></span>
              Canlı Piyasa Fiyatları
            </h2>
            <p className="text-xs text-slate-500 mt-0.5">
              Altınkaynak API üzerinden WebSocket ile anlık güncellenir.
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <div className="inline-flex rounded-xl bg-slate-100 p-1 text-xs font-semibold text-slate-600">
              <button
                onClick={() => setActiveTab("all")}
                className={`rounded-lg px-3 py-1.5 transition-all ${
                  activeTab === "all"
                    ? "bg-white text-slate-900 shadow-xs"
                    : "hover:text-slate-900"
                }`}
              >
                Tümü ({markets.length})
              </button>
              <button
                onClick={() => setActiveTab("currency")}
                className={`rounded-lg px-3 py-1.5 transition-all ${
                  activeTab === "currency"
                    ? "bg-white text-slate-900 shadow-xs"
                    : "hover:text-slate-900"
                }`}
              >
                Döviz ({markets.filter((m) => m.type === "currency").length})
              </button>
              <button
                onClick={() => setActiveTab("gold")}
                className={`rounded-lg px-3 py-1.5 transition-all ${
                  activeTab === "gold"
                    ? "bg-white text-slate-900 shadow-xs"
                    : "hover:text-slate-900"
                }`}
              >
                Altın ({markets.filter((m) => m.type === "gold").length})
              </button>
            </div>

            <div className="relative">
              <input
                type="text"
                placeholder="Ara... (USD, Gram, Dolar)"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-48 rounded-xl border border-slate-200 bg-slate-50 px-3 py-1.5 text-xs text-slate-800 outline-none transition-all focus:border-indigo-500 focus:bg-white focus:ring-2 focus:ring-indigo-100"
              />
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery("")}
                  className="absolute right-2 top-1.2 text-xs text-slate-400 hover:text-slate-600"
                >
                  ✕
                </button>
              )}
            </div>
          </div>
        </div>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="border-b border-slate-200/80 bg-slate-50/80 text-xs font-semibold uppercase tracking-wider text-slate-500">
              <th className="px-5 py-3.5">Sembol</th>
              <th className="px-5 py-3.5">Açıklama</th>
              <th className="px-5 py-3.5">Kategori</th>
              <th className="px-5 py-3.5 text-right">Alış Fiyatı</th>
              <th className="px-5 py-3.5 text-right">Satış Fiyatı</th>
              <th className="px-5 py-3.5 text-right">Anlık Değişim</th>
              <th className="px-5 py-3.5 text-right">Günlük Fark</th>
              <th className="px-5 py-3.5 text-right">Günlük Değişim</th>
              <th className="px-5 py-3.5 text-right">Son Güncelleme</th>
            </tr>
          </thead>

          <tbody className="divide-y divide-slate-100 text-sm">
            {filteredMarkets.length > 0 ? (
              filteredMarkets.map((market) => {
                const isUp = market.trend === "up" || market.changePercent > 0;
                const isDown = market.trend === "down" || market.changePercent < 0;

                const isInstantUp = market.instantTrend === "up";
                const isInstantDown = market.instantTrend === "down";

                return (
                  <tr
                    key={market.symbol}
                    className="transition-colors hover:bg-slate-50/80 group"
                  >
                    <td className="px-5 py-3.5 font-bold text-slate-900 group-hover:text-indigo-600 transition-colors">
                      {market.symbol}
                    </td>

                    <td className="px-5 py-3.5 text-slate-700 font-medium">
                      {market.name}
                    </td>

                    <td className="px-5 py-3.5">
                      <span
                        className={`inline-flex items-center rounded-md px-2.5 py-0.5 text-xs font-medium ${
                          market.type === "currency"
                            ? "bg-blue-50 text-blue-700 border border-blue-200/60"
                            : "bg-amber-50 text-amber-700 border border-amber-200/60"
                        }`}
                      >
                        {market.typeLabel}
                      </span>
                    </td>

                    <td className="px-5 py-3.5 text-right font-semibold text-slate-900 tabular-nums">
                      {market.buyPrice.toLocaleString("tr-TR", {
                        minimumFractionDigits: 2,
                        maximumFractionDigits: 4,
                      })}{" "}
                      ₺
                    </td>

                    <td className="px-5 py-3.5 text-right font-semibold text-slate-900 tabular-nums">
                      {market.sellPrice.toLocaleString("tr-TR", {
                        minimumFractionDigits: 2,
                        maximumFractionDigits: 4,
                      })}{" "}
                      ₺
                    </td>

                    <td className="px-5 py-3.5 text-right font-semibold tabular-nums">
                      <span
                        className={`inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-bold transition-all ${
                          isInstantUp
                            ? "bg-emerald-500 text-white shadow-xs"
                            : isInstantDown
                            ? "bg-rose-500 text-white shadow-xs"
                            : "bg-slate-100 text-slate-400 font-normal"
                        }`}
                      >
                        {isInstantUp && `▲ +${market.instantDiff ? Math.abs(market.instantDiff).toFixed(3) : "0"} ₺`}
                        {isInstantDown && `▼ -${market.instantDiff ? Math.abs(market.instantDiff).toFixed(3) : "0"} ₺`}
                        {!isInstantUp && !isInstantDown && "• Yatay"}
                      </span>
                    </td>

                    <td
                      className={`px-5 py-3.5 text-right font-semibold tabular-nums text-xs ${
                        isUp
                          ? "text-emerald-600"
                          : isDown
                          ? "text-rose-600"
                          : "text-slate-500"
                      }`}
                    >
                      {isUp ? "+" : ""}
                      {market.changeAmount.toLocaleString("tr-TR", {
                        minimumFractionDigits: 2,
                        maximumFractionDigits: 4,
                      })}{" "}
                      ₺
                    </td>

                    <td className="px-5 py-3.5 text-right font-semibold tabular-nums">
                      <span
                        className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-bold ${
                          isUp
                            ? "bg-emerald-100 text-emerald-700"
                            : isDown
                            ? "bg-rose-100 text-rose-700"
                            : "bg-slate-100 text-slate-600"
                        }`}
                      >
                        {isUp && "+"}
                        {market.changePercent}%
                      </span>
                    </td>

                    <td className="px-5 py-3.5 text-right text-xs text-slate-500 tabular-nums">
                      {market.updatedAt
                        ? new Date(market.updatedAt).toLocaleTimeString("tr-TR", {
                            hour: "2-digit",
                            minute: "2-digit",
                            second: "2-digit",
                          })
                        : "-"}
                    </td>
                  </tr>
                );
              })
            ) : (
              <tr>
                <td colSpan="9" className="py-12 text-center text-slate-500">
                  Aramaya uygun piyasa verisi bulunamadı.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export default MarketTable;