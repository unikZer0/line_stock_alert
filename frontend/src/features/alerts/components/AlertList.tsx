import { AlertCard } from "./AlertCard";
import type { StockAlert } from "../types/alert";

interface AlertListProps {
  alerts: StockAlert[];
  onEdit: (alert: StockAlert) => void;
  onDelete: (alert: StockAlert) => void;
  onRearm: (alert: StockAlert) => void;
}

export function AlertList({ alerts, onEdit, onDelete, onRearm }: AlertListProps) {
  return <section className="alert-list">{alerts.map((alert) => (
    <AlertCard alert={alert} key={alert.id} onEdit={onEdit} onDelete={onDelete} onRearm={onRearm} />
  ))}</section>;
}
