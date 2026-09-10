import type { CandleRange, StockCandleSeries } from "../types/stock";

const prefix = "alert-bot:candles:";
export function readStoredCandles(symbol: string, range: CandleRange): StockCandleSeries | null {
  try {
    const stored = JSON.parse(localStorage.getItem(`${prefix}${symbol}:${range}`) || "null") as { series?: StockCandleSeries } | null;
    return stored?.series ?? null;
  } catch { return null; }
}
export function storeCandles(series: StockCandleSeries) {
  try { localStorage.setItem(`${prefix}${series.symbol}:${series.range}`, JSON.stringify({ savedAt: Date.now(), series })); } catch { /* Network loading remains available. */ }
}
