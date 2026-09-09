import { EmptyState } from "../../../components/common/EmptyState";
import { ErrorState } from "../../../components/common/ErrorState";
import { LoadingState } from "../../../components/common/LoadingState";
import { useStockCandles } from "../hooks/useStockCandles";
import type { CandleRange } from "../types/stock";
import { CandlestickChart } from "./CandlestickChart";

const ranges: CandleRange[] = ["1D", "1W", "1M", "3M", "1Y", "MAX"];
export function StockHistoryChart({ symbol }: { symbol: string }) {
  const { series, range, setRange, loading, error } = useStockCandles(symbol);
  return <section className="card history" aria-labelledby="history-title">
    <div className="history-header"><div><p className="eyebrow">Historical prices</p><h2 id="history-title">{symbol} candles</h2>{series && <small>{series.interval} · {series.timezone}</small>}</div>
      <div className="range-selector" aria-label="Chart range">{ranges.map((option) => <button key={option} className={option === range ? "selected" : ""} onClick={() => setRange(option)}>{option}</button>)}</div>
    </div>
    {loading && <LoadingState />}{error && <ErrorState message={error} />}
    {!loading && !error && series?.candles.length === 0 && <EmptyState>No historical prices are available.</EmptyState>}
    {!loading && series && series.candles.length > 0 && <CandlestickChart candles={series.candles} />}
  </section>;
}
