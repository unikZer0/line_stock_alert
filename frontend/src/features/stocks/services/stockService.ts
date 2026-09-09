import { apiRequest } from "../../../services/api/apiClient";
import type { ApiPageResponse, ApiResponse } from "../../../types/api";
import type { CandleRange, Stock, StockCandleSeries, StockQuote } from "../types/stock";

export function getStocks(search: string, page: number) {
  return apiRequest<ApiPageResponse<Stock[]>>(
    `/stocks?search=${encodeURIComponent(search)}&page=${page}&limit=20`,
  );
}

export function getStockQuote(symbol: string) {
  return apiRequest<ApiResponse<StockQuote>>(`/stocks/${encodeURIComponent(symbol)}/quote`);
}

export function getStockCandles(symbol: string, range: CandleRange) {
  return apiRequest<ApiResponse<StockCandleSeries>>(`/stocks/${encodeURIComponent(symbol)}/candles?range=${range}`);
}
