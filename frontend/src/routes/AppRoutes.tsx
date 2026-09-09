import { AlertsPage } from "../features/alerts/pages/AlertsPage";
import { StockDetailPage } from "../features/stocks/pages/StockDetailPage";
import { StocksPage } from "../features/stocks/pages/StocksPage";

export function AppRoutes() {
  const stockMatch = window.location.pathname.match(/^\/stocks\/([^/]+)$/);

  if (stockMatch) {
    return <StockDetailPage symbol={decodeURIComponent(stockMatch[1]).toUpperCase()} />;
  }

  if (window.location.pathname === "/alerts") {
    return <AlertsPage />;
  }

  return <StocksPage />;
}
