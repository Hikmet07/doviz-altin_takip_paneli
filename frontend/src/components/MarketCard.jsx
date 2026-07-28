function MarketCard({ title, symbol, buyPrice, sellPrice, changePercent, trend, typeLabel }) {
  const isUp = trend === "up" || Number(changePercent) > 0;
  const isDown = trend === "down" || Number(changePercent) < 0;

  return (
    <div className="relative overflow-hidden rounded-2xl border border-slate-200 bg-white p-5 shadow-xs transition-all hover:shadow-md hover:-translate-y-0.5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <span className="text-xs font-bold uppercase tracking-wider text-slate-400">
            {symbol}
          </span>
          <h3 className="text-base font-bold text-slate-900 leading-tight">
            {title}
          </h3>
        </div>

        <span
          className={`inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-bold ${
            isUp
              ? "bg-emerald-100 text-emerald-700"
              : isDown
              ? "bg-rose-100 text-rose-700"
              : "bg-slate-100 text-slate-600"
          }`}
        >
          {isUp ? "▲ +" : isDown ? "▼ " : "• "}
          {changePercent}%
        </span>
      </div>

      <div className="mt-4 grid grid-cols-2 gap-3 border-t border-slate-100 pt-3">
        <div>
          <span className="text-xs text-slate-400 font-medium">Alış</span>
          <p className="text-sm font-bold text-slate-900 tabular-nums">
            {Number(buyPrice).toLocaleString("tr-TR", {
              minimumFractionDigits: 2,
              maximumFractionDigits: 4,
            })}{" "}
            ₺
          </p>
        </div>

        <div className="text-right">
          <span className="text-xs text-slate-400 font-medium">Satış</span>
          <p className="text-sm font-bold text-indigo-600 tabular-nums">
            {Number(sellPrice).toLocaleString("tr-TR", {
              minimumFractionDigits: 2,
              maximumFractionDigits: 4,
            })}{" "}
            ₺
          </p>
        </div>
      </div>
    </div>
  );
}

export default MarketCard;