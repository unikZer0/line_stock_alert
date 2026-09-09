import { useEffect, useState, type FormEvent } from "react";

import { createAlert } from "../services/alertService";
import type { AlertCondition } from "../types/alert";

interface AlertFormProps {
  symbol: string;
  initialPrice: number;
}

export function AlertForm({ symbol, initialPrice }: AlertFormProps) {
  const [condition, setCondition] = useState<AlertCondition>("ABOVE");
  const [targetPrice, setTargetPrice] = useState(String(initialPrice));
  const [submitting, setSubmitting] = useState(false);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");

  useEffect(() => setTargetPrice(String(initialPrice)), [initialPrice]);

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    setSubmitting(true);
    setNotice("");
    setError("");
    try {
      await createAlert({ symbol, condition, target_price: Number(targetPrice) });
      setNotice("Alert created. LINE will notify you once when it triggers.");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Could not create the alert.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form className="card form" onSubmit={handleSubmit}>
      <h2>Create price alert</h2>
      <label>Condition
        <select value={condition} onChange={(event) => setCondition(event.target.value as AlertCondition)}>
          <option value="ABOVE">ABOVE</option>
          <option value="BELOW">BELOW</option>
        </select>
      </label>
      <label>Target price (USD)
        <input type="number" min="0.01" step="0.01" required value={targetPrice} onChange={(event) => setTargetPrice(event.target.value)} />
      </label>
      <button type="submit" disabled={submitting}>{submitting ? "Creating…" : "Create alert"}</button>
      {notice && <p className="success">{notice}</p>}
      {error && <p className="error" role="alert">{error}</p>}
    </form>
  );
}
