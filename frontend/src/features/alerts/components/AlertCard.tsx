import type { StockAlert } from "../types/alert";

interface AlertCardProps {
  alert: StockAlert;
  onEdit: (alert: StockAlert) => void;
  onDelete: (alert: StockAlert) => void;
  onRearm: (alert: StockAlert) => void;
}

export function AlertCard({ alert, onEdit, onDelete, onRearm }: AlertCardProps) {
  return (
    <article className="card alert">
      <div>
        <strong>{alert.symbol}</strong>
        <span>{alert.condition} ${alert.target_price.toFixed(2)}</span>
      </div>
      <em className={alert.status.toLowerCase()}>{alert.status}</em>
      <div className="actions">
        <button disabled={alert.status !== "ACTIVE"} onClick={() => onEdit(alert)}>Edit</button>
        {alert.status === "TRIGGERED" && <button onClick={() => onRearm(alert)}>Re-arm</button>}
        <button className="danger" onClick={() => onDelete(alert)}>Delete</button>
      </div>
    </article>
  );
}
