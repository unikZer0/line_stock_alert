export type AlertCondition = "ABOVE" | "BELOW";
export type AlertStatus = "ACTIVE" | "TRIGGERED" | "DISABLED";

export interface StockAlert {
  id: string;
  symbol: string;
  condition: AlertCondition;
  target_price: number;
  status: AlertStatus;
  triggered_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAlertInput {
  symbol: string;
  condition: AlertCondition;
  target_price: number;
}

export interface UpdateAlertInput {
  condition: AlertCondition;
  target_price: number;
}
