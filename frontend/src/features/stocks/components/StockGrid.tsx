import { rememberStock } from "../services/stockSearchHistory";
import type { Stock } from "../types/stock";
export function StockGrid({ stocks }: { stocks: Stock[] }) {
  return <section className="stock-feed">{stocks.map((stock) => <a className="stock-row" href={`/stocks/${stock.symbol}`} onClick={() => rememberStock(stock)} key={stock.symbol}><div><b>{stock.symbol}</b><small>{stock.exchange || "US"}</small><span>{stock.name || "US stock"}</span></div><strong>›</strong></a>)}</section>;
}
