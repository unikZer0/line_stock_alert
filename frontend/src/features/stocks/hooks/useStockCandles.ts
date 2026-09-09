import { useEffect, useState } from "react";
import { readStoredCandles, storeCandles } from "../services/candleStorage";
import { getStockCandles } from "../services/stockService";
import type { CandleRange, StockCandleSeries } from "../types/stock";

export function useStockCandles(symbol: string) {
  const [range, setRange] = useState<CandleRange>("1D");
  const [series, setSeries] = useState<StockCandleSeries | null>(() => readStoredCandles(symbol, "1D"));
  const [loading, setLoading] = useState(() => readStoredCandles(symbol, "1D") === null);
  const [error, setError] = useState("");
  useEffect(() => {
    let cancelled = false;
    const stored = readStoredCandles(symbol, range);
    setSeries(stored); setLoading(stored === null); setError("");
    getStockCandles(symbol, range)
      .then((response) => { if (!cancelled) { setSeries(response.data); storeCandles(response.data); } })
      .catch((requestError: unknown) => { if (!cancelled && !stored) { setSeries(null); setError(requestError instanceof Error ? requestError.message : "Could not load historical prices."); } })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [symbol, range]);
  return { series, range, setRange, loading, error };
}
