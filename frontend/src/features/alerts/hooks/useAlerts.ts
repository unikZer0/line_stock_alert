import { useCallback, useEffect, useState } from "react";

import { deleteAlert, getAlerts, rearmAlert, updateAlert } from "../services/alertService";
import type { StockAlert } from "../types/alert";

export function useAlerts() {
  const [alerts, setAlerts] = useState<StockAlert[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const response = await getAlerts();
      setAlerts(response.data);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Could not load alerts.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const edit = async (alert: StockAlert, targetPrice: number) => {
    await updateAlert(alert.id, { condition: alert.condition, target_price: targetPrice });
    await load();
  };

  const remove = async (alert: StockAlert) => {
    await deleteAlert(alert.id);
    await load();
  };

  const rearm = async (alert: StockAlert) => {
    await rearmAlert(alert.id);
    await load();
  };

  return { alerts, loading, error, setError, edit, remove, rearm, reload: load };
}
