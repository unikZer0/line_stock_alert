import { apiRequest } from "../../../services/api/apiClient";
import type { ApiPageResponse, ApiResponse } from "../../../types/api";
import { API_BASE_URL } from "../../auth/services/sessionService";

export type AdminRecord = Record<string, unknown>;
export const getAdminDashboard = () => apiRequest<ApiResponse<AdminRecord>>("/admin/dashboard");
export const getAdminUsers = () => apiRequest<ApiPageResponse<AdminRecord[]>>("/admin/users?page=1&limit=50");
export const getAdminAlerts = () => apiRequest<ApiPageResponse<AdminRecord[]>>("/admin/alerts?page=1&limit=50");
export const getAdminStocks = () => apiRequest<ApiPageResponse<AdminRecord[]>>("/admin/stocks?page=1&limit=50");
export const getLineStats = () => apiRequest<ApiResponse<AdminRecord>>("/admin/line/stats");
export const getApplicationLogs = () => apiRequest<ApiPageResponse<AdminRecord[]>>("/admin/logs?page=1&limit=50");
export const getAuditLogs = () => apiRequest<ApiPageResponse<AdminRecord[]>>("/admin/audit-logs?page=1&limit=50");

export async function adminLogin(email: string, password: string) {
  const response = await fetch(`${API_BASE_URL}/auth/login`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email, password }) });
  const result = await response.json() as ApiResponse<{ access_token: string; refresh_token: string }> & { error?: { message?: string } };
  if (!response.ok) throw new Error(result.error?.message || "Administrator login failed.");
  return result.data;
}
