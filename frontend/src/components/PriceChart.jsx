import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
  Legend,
} from "recharts";

function downsampleData(data, range) {
  if (!Array.isArray(data) || data.length === 0) return [];

  if (range === "1d") {
    return data;
  }

  const dayGroups = new Map();

  data.forEach((item) => {
    const d = item.createdAt ? new Date(item.createdAt) : new Date();
    const dateKey = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;

    if (!dayGroups.has(dateKey)) {
      dayGroups.set(dateKey, []);
    }
    dayGroups.get(dateKey).push(item);
  });

  const sampled = [];

  dayGroups.forEach((dayItems) => {
    if (range === "30d") {
      sampled.push(dayItems[dayItems.length - 1]);
    } else if (range === "7d") {
      if (dayItems.length <= 2) {
        sampled.push(...dayItems);
      } else {
        const mid = Math.floor(dayItems.length / 2);
        sampled.push(dayItems[mid]);
        sampled.push(dayItems[dayItems.length - 1]);
      }
    }
  });

  return sampled;
}

function PriceChart({ data, range = "30d", title = "Fiyat Grafiği" }) {
  const sampledData = downsampleData(data, range);

  const formattedData = sampledData.map((item, index) => {
    const d = item.createdAt ? new Date(item.createdAt) : new Date();

    let displayTick = "";
    if (range === "1d") {
      displayTick = d.toLocaleTimeString("tr-TR", {
        hour: "2-digit",
        minute: "2-digit",
      });
    } else {
      displayTick = d.toLocaleDateString("tr-TR", {
        day: "2-digit",
        month: "2-digit",
      });
    }

    return {
      ...item,
      chartKey: item.id || `${item.createdAt}-${index}`,
      displayTick,
      fullDateTime: d.toLocaleString("tr-TR", {
        day: "2-digit",
        month: "2-digit",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      }),
    };
  });

  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-xs">
      <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-xl font-bold text-slate-900">{title}</h2>
          <p className="text-xs text-slate-500 mt-0.5">
            Zaman serisi fiyat değişimi (Alış & Satış)
          </p>
        </div>

        {data.length > 0 && (
          <div className="rounded-lg bg-slate-50 border border-slate-100 px-3 py-1.5 text-xs text-slate-600 font-medium">
            Gösterilen: <span className="font-bold text-indigo-600">{formattedData.length}</span> / Toplam {data.length} kayıt
          </div>
        )}
      </div>

      {data.length === 0 ? (
        <div className="flex h-80 flex-col items-center justify-center rounded-xl bg-slate-50 border border-dashed border-slate-200 p-6 text-center text-slate-400">
          <p className="text-sm font-medium">Bu ürün için henüz geçmiş grafik verisi bulunamadı.</p>
          <p className="text-xs mt-1">Backend veritabanında veri biriktikçe grafikler görüntülenecektir.</p>
        </div>
      ) : (
        <div className="h-96 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={formattedData} margin={{ top: 10, right: 20, left: 10, bottom: 0 }}>
              <defs>
                <linearGradient id="colorBuy" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#10b981" stopOpacity={0.25} />
                  <stop offset="95%" stopColor="#10b981" stopOpacity={0.0} />
                </linearGradient>
                <linearGradient id="colorSell" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#6366f1" stopOpacity={0.25} />
                  <stop offset="95%" stopColor="#6366f1" stopOpacity={0.0} />
                </linearGradient>
              </defs>

              <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />

              <XAxis
                dataKey="displayTick"
                stroke="#94a3b8"
                fontSize={11}
                tickLine={false}
                minTickGap={15}
              />

              <YAxis
                domain={["auto", "auto"]}
                stroke="#94a3b8"
                fontSize={11}
                tickLine={false}
                tickFormatter={(val) => `${val.toLocaleString("tr-TR")}`}
              />

              <Tooltip
                labelFormatter={(_, items) =>
                  items && items[0] ? items[0].payload.fullDateTime : ""
                }
                contentStyle={{
                  backgroundColor: "#ffffff",
                  borderColor: "#e2e8f0",
                  borderRadius: "12px",
                  boxShadow: "0 4px 12px rgba(0, 0, 0, 0.08)",
                  fontSize: "12px",
                }}
                formatter={(value) => [
                  `${Number(value).toLocaleString("tr-TR", {
                    minimumFractionDigits: 2,
                    maximumFractionDigits: 4,
                  })} ₺`,
                ]}
              />

              <Legend verticalAlign="top" height={36} />

              <Area
                type="monotone"
                dataKey="buyPrice"
                name="Alış Fiyatı"
                stroke="#10b981"
                strokeWidth={2.5}
                fillOpacity={1}
                fill="url(#colorBuy)"
              />

              <Area
                type="monotone"
                dataKey="sellPrice"
                name="Satış Fiyatı"
                stroke="#6366f1"
                strokeWidth={2.5}
                fillOpacity={1}
                fill="url(#colorSell)"
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}

export default PriceChart;