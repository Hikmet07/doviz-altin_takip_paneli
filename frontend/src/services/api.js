const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL || "https://doviz-altin-takip-paneli.onrender.com";

function convertRangeToDays(range) {
  if (range === "1d") return 1;
  if (range === "7d") return 7;
  if (range === "30d") return 30;

  return 30;
}

export function normalizePriceRecord(record) {
  const changeAmt = Number(record.fark) || 0;
  const changePct = Number(record.gunluk_degisim_yuzde) || 0;

  return {
    id: record.id,
    symbol: record.kod,
    name: record.aciklama ? record.aciklama.trim() : record.kod,
    type: record.kategori, // "currency" or "gold"
    typeLabel: record.kategori === "currency" ? "Döviz" : record.kategori === "gold" ? "Altın" : "Diğer",
    buyPrice: Number(record.alis) || 0,
    sellPrice: Number(record.satis) || 0,
    changeAmount: changeAmt,
    changePercent: changePct,
    trend: changePct > 0 ? "up" : changePct < 0 ? "down" : "same",
    updatedAt: record.guncelleme_zamani,
    createdAt: record.created_at,
  };
}

export async function getMarkets() {
  const response = await fetch(`${API_BASE_URL}/api/prices/latest`);

  if (!response.ok) {
    throw new Error("Güncel fiyat verileri alınamadı.");
  }

  const data = await response.json();

  return Array.isArray(data) ? data.map(normalizePriceRecord) : [];
}

export async function getCurrencyPrices() {
  const response = await fetch(`${API_BASE_URL}/api/prices/latest/currency`);

  if (!response.ok) {
    throw new Error("Döviz fiyatları alınamadı.");
  }

  const data = await response.json();

  return Array.isArray(data) ? data.map(normalizePriceRecord) : [];
}

export async function getGoldPrices() {
  const response = await fetch(`${API_BASE_URL}/api/prices/latest/gold`);

  if (!response.ok) {
    throw new Error("Altın fiyatları alınamadı.");
  }

  const data = await response.json();

  return Array.isArray(data) ? data.map(normalizePriceRecord) : [];
}

export async function getHistory(symbol, range = "30d") {
  const days = convertRangeToDays(range);

  const response = await fetch(
    `${API_BASE_URL}/api/prices/history/${symbol}?days=${days}`
  );

  if (!response.ok) {
    throw new Error("Geçmiş fiyat verileri alınamadı.");
  }

  const data = await response.json();

  return Array.isArray(data) ? data.map(normalizePriceRecord) : [];
}

export async function getSymbols() {
  const markets = await getMarkets();

  return markets.map((market) => ({
    symbol: market.symbol,
    name: market.name,
    type: market.type,
  }));
}