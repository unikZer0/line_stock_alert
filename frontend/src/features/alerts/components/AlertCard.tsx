import type { StockAlert } from "../types/alert";

interface Props { alert: StockAlert; onEdit: (alert: StockAlert) => void; onDelete: (alert: StockAlert) => void; onRearm: (alert: StockAlert) => void; }
export function AlertCard({ alert, onEdit, onDelete, onRearm }: Props) {
  const triggered = alert.status === "TRIGGERED";
  const openDetail = () => window.location.assign(`/stocks/${encodeURIComponent(alert.symbol)}`);
  return <article className={`alert-terminal ${triggered ? "is-triggered" : ""}`} role="link" tabIndex={0}
    aria-label={`Open ${alert.symbol} stock details`}
    onClick={(event) => { if (!(event.target instanceof Element && event.target.closest("button"))) openDetail(); }}
    onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); openDetail(); } }}>
    <div className="alert-top"><span className="ticker-avatar">{alert.symbol.slice(0,2)}</span><div><h2>{alert.symbol}</h2><small>US EQUITY</small></div><em className={alert.status.toLowerCase()}>{alert.status}</em></div>
    <div className="trigger-matrix"><span>TRIGGER TARGET <b>{alert.condition} ${alert.target_price.toFixed(2)}</b></span><span>CREATED <b>{new Date(alert.created_at).toLocaleDateString()}</b></span>{alert.triggered_at && <span>TRIGGERED <b>{new Date(alert.triggered_at).toLocaleString()}</b></span>}</div>
    <div className="alert-footer"><span><i /> {triggered ? "TRIGGERED · PAUSED" : "POLLING · 15s SYNC"}</span><div>{!triggered && <button onClick={() => onEdit(alert)}>⌁ Target</button>}{triggered && <button onClick={() => onRearm(alert)}>↻ Re-arm</button>}<button aria-label={`Delete ${alert.symbol} alert`} onClick={() => onDelete(alert)}>⌫</button></div></div>
  </article>;
}
