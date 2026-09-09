import { useCallback, useEffect, useState } from "react";

import { getStocks } from "../services/stockService";
import type { Stock } from "../types/stock";

export function useStocks() {
  const [query, setQuery] = useState("");
  const [stocks, setStocks] = useState<Stock[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async (nextPage: number, append: boolean, search: string) => {
    setLoading(true);
    setError("");
    try {
      const response = await getStocks(search, nextPage);
      setStocks((current) => append ? [...current, ...response.data] : response.data);
      setPage(nextPage);
      setTotal(response.meta.total);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Could not load stocks.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load(1, false, "");
  }, [load]);

  return { query, setQuery, stocks, page, total, loading, error, load };
}
