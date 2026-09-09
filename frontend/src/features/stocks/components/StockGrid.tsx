import type { Stock } from "../types/stock";

interface StockGridProps {
  stocks: Stock[];
}

export function StockGrid({ stocks }: StockGridProps) {
  return (
    <section className="grid">
      {stocks.map((stock) => (
        <a className="card stock" href={`/stocks/${stock.symbol}`} key={stock.symbol}>
          <strong>{stock.symbol}</strong>
          <span>{stock.name || "US stock"}</span>
          <small>{stock.exchange} · {stock.currency || "USD"}</small>
        </a>
      ))}
    </section>
  );
}
