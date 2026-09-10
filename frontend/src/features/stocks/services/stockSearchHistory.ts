import type { Stock } from "../types/stock";

const storageKey = "stock-alert:recent-stocks";

export function getRecentStocks(): Stock[] {
  try {
    const value = JSON.parse(localStorage.getItem(storageKey) || "[]") as Stock[];
    return Array.isArray(value) ? value.slice(0, 6) : [];
  } catch {
    return [];
  }
}

export function rememberStock(stock: Stock) {
  const recent = getRecentStocks().filter((item) => item.symbol !== stock.symbol);
  localStorage.setItem(storageKey, JSON.stringify([stock, ...recent].slice(0, 6)));
}

export function clearRecentStocks() {
  localStorage.removeItem(storageKey);
}
