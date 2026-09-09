import { useEffect, useState, type FormEvent, type KeyboardEvent } from "react";
import { clearRecentStocks, getRecentStocks, rememberStock } from "../services/stockSearchHistory";
import type { Stock } from "../types/stock";

interface StockSearchProps { query: string; suggestions: Stock[]; loading: boolean; onQueryChange: (query: string) => void; onSearch: () => void; }
export function StockSearch({ query, suggestions, loading, onQueryChange, onSearch }: StockSearchProps) {
  const [focused, setFocused] = useState(false);
  const [recent, setRecent] = useState(getRecentStocks);
  const [activeIndex, setActiveIndex] = useState(0);
  const results = query.trim() ? (loading ? [] : suggestions.slice(0, 6)) : recent;
  const openStock = (stock: Stock) => { rememberStock(stock); window.location.assign(`/stocks/${encodeURIComponent(stock.symbol)}`); };
  const submit = (event: FormEvent) => { event.preventDefault(); if (focused && results[activeIndex]) openStock(results[activeIndex]); else onSearch(); };
  const clearHistory = () => { clearRecentStocks(); setRecent([]); };
  useEffect(() => setActiveIndex(0), [query, suggestions]);
  const handleKeys = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown") { event.preventDefault(); setActiveIndex((index) => Math.min(results.length - 1, index + 1)); }
    if (event.key === "ArrowUp") { event.preventDefault(); setActiveIndex((index) => Math.max(0, index - 1)); }
    if (event.key === "Escape") setFocused(false);
  };
  return <div className="search-shell"><form className="search" onSubmit={submit}>
    <input aria-label="Search US stocks" autoComplete="off" value={query} onKeyDown={handleKeys} onFocus={() => { setRecent(getRecentStocks()); setFocused(true); }} onBlur={() => window.setTimeout(() => setFocused(false), 120)} onChange={(event) => onQueryChange(event.target.value)} placeholder="Search ticker or company…" />
    <button type="submit">{loading ? "…" : "Search"}</button>
  </form>{focused && (results.length > 0 || query.trim()) && <div className="search-suggestions">
    <div className="suggestion-heading"><span>{query.trim() ? "BEST MATCHES" : "RECENT SEARCHES"}</span>{!query.trim() && recent.length > 0 && <button onMouseDown={(event) => event.preventDefault()} onClick={clearHistory}>Clear</button>}</div>
    {results.map((stock, index) => <button className={`suggestion-row ${index === activeIndex ? "is-active" : ""}`} type="button" key={stock.symbol} onMouseEnter={() => setActiveIndex(index)} onMouseDown={(event) => event.preventDefault()} onClick={() => openStock(stock)}><span><b>{stock.symbol}</b><small>{stock.exchange || "US"}</small><em>{stock.name || "US stock"}</em></span><strong>›</strong></button>)}
    {query.trim() && loading && <p>Finding the best ticker matches…</p>}
    {query.trim() && !loading && results.length === 0 && <p>No matching symbols</p>}
  </div>}</div>;
}
