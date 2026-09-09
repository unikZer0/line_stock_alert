export interface Stock {
  symbol: string;
  name: string;
  exchange: string;
  currency: string;
}

export interface StockQuote extends Stock {
  price: number;
  change: number;
  change_percent: number;
  open: number;
  high: number;
  low: number;
  previous_close: number;
  market_status: string;
  market_status_until: string;
  updated_at: string;
  cached: boolean;
}

export type CandleRange = "1D" | "1W" | "1M" | "3M" | "1Y" | "MAX";
export interface StockCandle { timestamp: string; open: number; high: number; low: number; close: number; volume: number; }
export interface StockCandleSeries {
  symbol: string; range: CandleRange; interval: string; currency: string; timezone: string;
  candles: StockCandle[]; cached: boolean;
}
