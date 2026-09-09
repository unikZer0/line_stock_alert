import { useCallback, useEffect, useRef, useState } from "react";

import { getStocks } from "../services/stockService";
import type { Stock } from "../types/stock";

export function useStocks() {
  const [query, setQuery] = useState("");
  const [stocks, setStocks] = useState<Stock[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const requestNumber = useRef(0);

  const load = useCallback(async (nextPage: number, append: boolean, search: string) => {
    const request = ++requestNumber.current;
    setLoading(true);
    setError("");
    try {
      const response = await getStocks(search, nextPage);
      if (request !== requestNumber.current) return;
      setStocks((current) => append ? [...current, ...response.data] : response.data);
      setPage(nextPage);
      setTotal(response.meta.total);
    } catch (requestError) {
      if (request === requestNumber.current) setError(requestError instanceof Error ? requestError.message : "Could not load stocks.");
    } finally {
      if (request === requestNumber.current) setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => void load(1, false, query), 120);
    return () => window.clearTimeout(timer);
  }, [load, query]);

  return { query, setQuery, stocks, page, total, loading, error, load };
}
