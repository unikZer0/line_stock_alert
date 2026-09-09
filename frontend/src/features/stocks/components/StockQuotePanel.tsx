import type { StockQuote } from "../types/stock";
import { MarketStatus } from "./MarketStatus";

interface StockQuotePanelProps {
  quote: StockQuote;
}

export function StockQuotePanel({ quote }: StockQuotePanelProps) {
  return (
    <section className="card detail">
      <p className="eyebrow">{quote.symbol}</p>
      <h1>{quote.name || quote.symbol}</h1>
      <div className="price">${quote.price.toFixed(2)}</div>
      <p className={quote.change >= 0 ? "positive" : "negative"}>
        {quote.change >= 0 ? "+" : ""}{quote.change.toFixed(2)} ({quote.change_percent.toFixed(2)}%)
      </p>
      <MarketStatus status={quote.market_status} until={quote.market_status_until} />
      <div className="stats">
        <span>Open <b>${quote.open.toFixed(2)}</b></span>
        <span>High <b>${quote.high.toFixed(2)}</b></span>
        <span>Low <b>${quote.low.toFixed(2)}</b></span>
        <span>Previous <b>${quote.previous_close.toFixed(2)}</b></span>
      </div>
    </section>
  );
}
