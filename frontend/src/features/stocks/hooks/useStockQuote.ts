import { useEffect, useState } from "react";

import { getStockQuote } from "../services/stockService";
import type { StockQuote } from "../types/stock";

export function useStockQuote(symbol: string) {
  const [quote, setQuote] = useState<StockQuote | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    setLoading(true);
    getStockQuote(symbol)
      .then((response) => setQuote(response.data))
      .catch((requestError: unknown) => {
        setError(requestError instanceof Error ? requestError.message : "Could not load this quote.");
      })
      .finally(() => setLoading(false));
  }, [symbol]);

  return { quote, loading, error };
}
