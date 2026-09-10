import { clearAccessToken, getAccessToken, startLineLogin, API_BASE_URL } from "../../features/auth/services/sessionService";
import type { ApiErrorResponse } from "../../types/api";

export async function apiRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const accessToken = getAccessToken();

  if (!accessToken) {
    startLineLogin();
    throw new Error("Opening LINE Login…");
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${accessToken}`,
      ...options.headers,
    },
  });

  if (response.status === 401) {
    clearAccessToken();
    startLineLogin();
    throw new Error("Your session expired. Opening LINE Login…");
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const body = (await response.json()) as T | ApiErrorResponse;
  if (!response.ok) {
    const error = body as ApiErrorResponse;
    throw new Error(error.error?.message || "Request failed.");
  }

  return body as T;
}
