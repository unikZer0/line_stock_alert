import { useEffect } from "react";
import { EmptyState } from "../../../components/common/EmptyState";
import { ErrorState } from "../../../components/common/ErrorState";
import { LoadingState } from "../../../components/common/LoadingState";
import { useAlerts } from "../hooks/useAlerts";
import type { StockAlert } from "../types/alert";
import { AlertList } from "./AlertList";

export function SymbolAlerts({ symbol, refreshKey }: { symbol: string; refreshKey: number }) {
  const { alerts, loading, error, setError, edit, remove, rearm, reload } = useAlerts();
  const matching = alerts.filter((alert) => alert.symbol === symbol);
  useEffect(() => { if (refreshKey > 0) void reload(); }, [refreshKey, reload]);
  const changeTarget = async (alert: StockAlert) => {
    const value = window.prompt(`New target price for ${alert.symbol}`, String(alert.target_price));
    if (!value) return;
    const target = Number(value);
    if (!Number.isFinite(target) || target <= 0) { setError("Target price must be greater than zero."); return; }
    try { await edit(alert, target); } catch (cause) { setError(cause instanceof Error ? cause.message : "Could not update the alert."); }
  };
  const deleteCurrent = async (alert: StockAlert) => { if (!window.confirm(`Delete the ${alert.symbol} alert?`)) return; try { await remove(alert); } catch (cause) { setError(cause instanceof Error ? cause.message : "Could not delete the alert."); } };
  const rearmCurrent = async (alert: StockAlert) => { try { await rearm(alert); } catch (cause) { setError(cause instanceof Error ? cause.message : "Could not re-arm the alert."); } };
  return <section className="symbol-alerts">
    <div className="symbol-alerts-heading"><div><p className="eyebrow">Alert registry</p><h2>Current alerts for {symbol}</h2></div><b>{matching.length}</b></div>
    {loading && <LoadingState />}{error && <ErrorState message={error} />}
    {!loading && !error && matching.length === 0 && <EmptyState>No alerts for this stock yet.</EmptyState>}
    <AlertList alerts={matching} onEdit={(alert) => void changeTarget(alert)} onDelete={(alert) => void deleteCurrent(alert)} onRearm={(alert) => void rearmCurrent(alert)} />
  </section>;
}
