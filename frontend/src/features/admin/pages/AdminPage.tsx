import { useEffect, useMemo, useState } from "react";
import { ErrorState } from "../../../components/common/ErrorState";
import { LoadingState } from "../../../components/common/LoadingState";
import { AdminLayout } from "../components/AdminLayout";
import { AdminTable } from "../components/AdminTable";
import { getAdminAlerts, getAdminDashboard, getAdminStocks, getAdminUsers, getApplicationLogs, getAuditLogs, getLineStats, type AdminRecord } from "../services/adminService";

type Config = { title: string; subtitle: string; load: () => Promise<unknown>; columns?: string[] };
const configs: Record<string, Config> = {
  "/admin/users": { title: "User Management", subtitle: "IDENTITY, LINE CONNECTION & ACCESS CONTROL", load: getAdminUsers, columns: ["display_name","email","role","status","line_connected","created_at"] },
  "/admin/alerts": { title: "Alerts Management & Surveillance", subtitle: "REAL-TIME TRIGGER ENGINE REGISTRY", load: getAdminAlerts, columns: ["symbol","user_email","condition","target_price","status","triggered_at","created_at"] },
  "/admin/stocks": { title: "Stocks Directory", subtitle: "PROVIDER SYMBOLS & ALERT COVERAGE", load: getAdminStocks, columns: ["symbol","name","exchange","provider","status","last_quote_price","active_alerts","updated_at"] },
  "/admin/audit-logs": { title: "Administrative Audit Trail & Security Ledger", subtitle: "IMMUTABLE PRIVILEGED ACTION HISTORY", load: getAuditLogs, columns: ["created_at","action","target_type","target_id","admin_user_id","ip_address","details"] },
};

export function AdminPage() {
  const path = window.location.pathname;
  const config = configs[path] ?? (path === "/admin/line" ? { title: "LINE Monitoring & Worker Logs", subtitle: "DELIVERY PIPELINE TELEMETRY", load: getApplicationLogs, columns: ["created_at","level","service","code","message","details"] } : { title: "Dashboard & System Telemetry", subtitle: "PLATFORM HEALTH & OPERATING METRICS", load: getAdminDashboard });
  const [rows, setRows] = useState<AdminRecord[]>([]), [summary, setSummary] = useState<AdminRecord | null>(null), [loading, setLoading] = useState(true), [error, setError] = useState("");
  const [search, setSearch] = useState("");
  useEffect(() => { setLoading(true); const loader = path === "/admin/line" ? Promise.all([getLineStats(), getApplicationLogs()]) : config.load();
    loader.then((response: unknown) => { if (Array.isArray(response)) { const [stats, logs] = response as [{ data: AdminRecord }, { data: AdminRecord[] }]; setSummary(stats.data); setRows(logs.data); return; } const result = response as { data: AdminRecord | AdminRecord[] }; if (Array.isArray(result.data)) setRows(result.data); else setSummary(result.data); }).catch((cause: unknown) => setError(cause instanceof Error ? cause.message : "Could not load admin data.")).finally(() => setLoading(false));
  }, [path]);
  const filtered = useMemo(() => rows.filter((row) => JSON.stringify(row).toLowerCase().includes(search.toLowerCase())), [rows, search]);
  const metrics = summary ? Object.entries(summary).flatMap(([group, value]) => typeof value === "object" && value ? Object.entries(value).filter(([, item]) => typeof item !== "object").map(([key,item]) => [`${group} ${key}`, item]) : [[group,value]]).slice(0,8) : [];
  return <AdminLayout><div className="admin-context"><span>ADMIN / {path.split("/").pop()?.toUpperCase() || "DASHBOARD"}</span><i /> SYSTEM HEALTHY</div>
    <section className="admin-title"><div><h1>{config.title}</h1><p>{config.subtitle}</p></div><button onClick={() => window.location.reload()}>↻ Refresh</button></section>
    {loading && <LoadingState />}{error && <ErrorState message={error} />}
    {metrics.length > 0 && <section className="admin-metrics">{metrics.map(([label,value]) => <article key={String(label)}><small>{String(label).replaceAll("_"," ")}</small><b>{String(value)}</b><span>LIVE DATABASE</span></article>)}</section>}
    {config.columns && <><div className="admin-filters"><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search current records…" /><span>{filtered.length} RECORDS</span></div><AdminTable rows={filtered} columns={config.columns} /></>}
  </AdminLayout>;
}
