import { useEffect, useState } from "react";
import { getStockCandles } from "../services/stockService";
import type { CandleRange, StockCandleSeries } from "../types/stock";

export function useStockCandles(symbol: string) {
  const [range, setRange] = useState<CandleRange>("1D");
  const [series, setSeries] = useState<StockCandleSeries | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  useEffect(() => {
    let cancelled = false;
    setLoading(true); setError("");
    getStockCandles(symbol, range)
      .then((response) => { if (!cancelled) setSeries(response.data); })
      .catch((requestError: unknown) => { if (!cancelled) { setSeries(null); setError(requestError instanceof Error ? requestError.message : "Could not load historical prices."); } })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [symbol, range]);
  return { series, range, setRange, loading, error };
}
