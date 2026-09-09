import { EmptyState } from "../../../components/common/EmptyState";
import { ErrorState } from "../../../components/common/ErrorState";
import { LoadingState } from "../../../components/common/LoadingState";
import { PageLayout } from "../../../components/layout/PageLayout";
import { AlertList } from "../components/AlertList";
import { useAlerts } from "../hooks/useAlerts";
import type { StockAlert } from "../types/alert";

export function AlertsPage() {
  const { alerts, loading, error, setError, edit, remove, rearm } = useAlerts();

  const handleEdit = async (alert: StockAlert) => {
    const target = window.prompt(`New target price for ${alert.symbol}`, String(alert.target_price));
    if (!target) return;
    try {
      await edit(alert, Number(target));
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Could not update the alert.");
    }
  };

  const handleDelete = async (alert: StockAlert) => {
    if (!window.confirm(`Delete the ${alert.symbol} alert?`)) return;
    try {
      await remove(alert);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Could not delete the alert.");
    }
  };

  const handleRearm = async (alert: StockAlert) => {
    try {
      await rearm(alert);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Could not re-arm the alert.");
    }
  };

  return (
    <PageLayout>
      <section className="hero">
        <p className="eyebrow">LINE notifications</p>
        <h1>My alerts</h1>
        <p>Triggered alerts pause until price moves away from the target and you re-arm them.</p>
      </section>
      {loading && <LoadingState />}
      {error && <ErrorState message={error} />}
      {!loading && !error && alerts.length === 0 && <EmptyState>No alerts yet. Open Stocks to create one.</EmptyState>}
      <AlertList
        alerts={alerts}
        onEdit={(alert) => void handleEdit(alert)}
        onDelete={(alert) => void handleDelete(alert)}
        onRearm={(alert) => void handleRearm(alert)}
      />
    </PageLayout>
  );
}
