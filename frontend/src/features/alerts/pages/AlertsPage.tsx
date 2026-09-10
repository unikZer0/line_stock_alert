import { useMemo, useState } from "react";
import { EmptyState } from "../../../components/common/EmptyState";
import { ErrorState } from "../../../components/common/ErrorState";
import { LoadingState } from "../../../components/common/LoadingState";
import { PageLayout } from "../../../components/layout/PageLayout";
import { GlobalMarketStatus } from "../../stocks/components/GlobalMarketStatus";
import { AlertList } from "../components/AlertList";
import { useAlerts } from "../hooks/useAlerts";
import type { AlertStatus, StockAlert } from "../types/alert";

type Filter = "ALL" | AlertStatus;
export function AlertsPage() {
  const { alerts, loading, error, setError, edit, remove, rearm } = useAlerts();
  const [filter, setFilter] = useState<Filter>("ALL");
  const [search, setSearch] = useState("");
  const filtered = useMemo(() => {
    const query = search.trim().toUpperCase();
    return alerts.filter((alert) => {
      const matchesFilter = filter === "ALL" || alert.status === filter;
      const matchesSearch = !query || alert.symbol.includes(query) || alert.condition.includes(query) || alert.status.includes(query) || String(alert.target_price).includes(query);
      return matchesFilter && matchesSearch;
    });
  }, [alerts, filter, search]);
  const count = (status: AlertStatus) => alerts.filter((alert) => alert.status === status).length;
  const handleEdit = async (alert: StockAlert) => { const target = window.prompt(`New target price for ${alert.symbol}`, String(alert.target_price)); if (!target) return; try { await edit(alert, Number(target)); } catch (cause) { setError(cause instanceof Error ? cause.message : "Could not update the alert."); } };
  const handleDelete = async (alert: StockAlert) => { if (!window.confirm(`Delete the ${alert.symbol} alert?`)) return; try { await remove(alert); } catch (cause) { setError(cause instanceof Error ? cause.message : "Could not delete the alert."); } };
  const handleRearm = async (alert: StockAlert) => { try { await rearm(alert); } catch (cause) { setError(cause instanceof Error ? cause.message : "Could not re-arm the alert."); } };
  return <PageLayout>
    <div className="telemetry"><span><i /> MARKET MONITOR</span><b>WORKER: LIVE RUNNING</b></div>
    <GlobalMarketStatus />
    <section className="page-title"><div><h1>My Price Alerts</h1><p>REAL-TIME LINE PUSH TRIGGERS</p></div><a className="primary-link" href="/stocks">＋ New Alert</a></section>
    <section className="counter-grid"><div><small>ACTIVE</small><b>{count("ACTIVE").toString().padStart(2,"0")}</b><span>Polling 15s</span></div><div><small>TRIGGERED</small><b>{count("TRIGGERED").toString().padStart(2,"0")}</b><span>Alert history</span></div><div><small>LINE PUSH</small><b>LIVE</b><span>Connected</span></div></section>
    <div className="alert-search"><span aria-hidden="true">⌕</span><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search alerts by ticker, status or target…" aria-label="Search my alerts" />{search && <button onClick={() => setSearch("")} aria-label="Clear alert search">×</button>}</div>
    <div className="filter-tabs">{(["ALL","ACTIVE","TRIGGERED"] as Filter[]).map((value) => <button className={filter === value ? "selected" : ""} onClick={() => setFilter(value)} key={value}>{value} ({value === "ALL" ? alerts.length : count(value)})</button>)}</div>
    {loading && <LoadingState />}{error && <ErrorState message={error} />}
    {!loading && !error && filtered.length === 0 && <EmptyState>{search ? `No alerts match “${search}”.` : "No alerts in this view."}</EmptyState>}
    <AlertList alerts={filtered} onEdit={(a) => void handleEdit(a)} onDelete={(a) => void handleDelete(a)} onRearm={(a) => void handleRearm(a)} />
    <div className="line-relay"><span className="line-logo">LINE</span><div><b>Notification relay active</b><small>Alerts dispatch directly to your LINE feed</small></div><i /></div>
  </PageLayout>;
}
