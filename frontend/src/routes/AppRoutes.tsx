import { AlertsPage } from "../features/alerts/pages/AlertsPage";
import { StockDetailPage } from "../features/stocks/pages/StockDetailPage";
import { StocksPage } from "../features/stocks/pages/StocksPage";
import { AdminPage } from "../features/admin/pages/AdminPage";
import { AdminLoginPage } from "../features/admin/pages/AdminLoginPage";
import { getAccessToken } from "../features/auth/services/sessionService";

export function AppRoutes() {
  if (window.location.pathname === "/admin/login") return <AdminLoginPage />;
  if (window.location.pathname.startsWith("/admin") && !getAccessToken()) { window.location.replace("/admin/login"); return null; }
  if (window.location.pathname === "/admin" || window.location.pathname.startsWith("/admin/")) return <AdminPage />;
  const stockMatch = window.location.pathname.match(/^\/stocks\/([^/]+)$/);

  if (stockMatch) {
    return <StockDetailPage symbol={decodeURIComponent(stockMatch[1]).toUpperCase()} />;
  }

  if (window.location.pathname === "/alerts") {
    return <AlertsPage />;
  }

  return <StocksPage />;
}
