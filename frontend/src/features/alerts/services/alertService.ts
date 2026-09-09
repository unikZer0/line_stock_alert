import { apiRequest } from "../../../services/api/apiClient";
import type { ApiResponse } from "../../../types/api";
import type { CreateAlertInput, StockAlert, UpdateAlertInput } from "../types/alert";

export function getAlerts() {
  return apiRequest<ApiResponse<StockAlert[]>>("/alerts");
}

export function createAlert(input: CreateAlertInput) {
  return apiRequest<ApiResponse<StockAlert>>("/alerts", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateAlert(id: string, input: UpdateAlertInput) {
  return apiRequest<ApiResponse<StockAlert>>(`/alerts/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export function deleteAlert(id: string) {
	return apiRequest<void>(`/alerts/${id}`, { method: "DELETE" });
}

export function rearmAlert(id: string) {
  return apiRequest<ApiResponse<StockAlert>>(`/alerts/${id}/rearm`, { method: "POST" });
}
