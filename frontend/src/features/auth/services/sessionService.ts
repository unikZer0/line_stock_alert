const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "/api/v1";
const ACCESS_TOKEN_KEY = "access_token";
const REFRESH_TOKEN_KEY = "refresh_token";
const RETURN_TO_KEY = "return_to";

export function consumeLineCallback() {
  if (window.location.pathname !== "/auth/callback") {
    return;
  }

  const data = new URLSearchParams(window.location.hash.slice(1));
  const accessToken = data.get("access_token");
  const refreshToken = data.get("refresh_token");

  if (accessToken) {
    localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  }
  if (refreshToken) {
    localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  }

  window.history.replaceState({}, "", sessionStorage.getItem(RETURN_TO_KEY) || "/stocks");
}

export function getAccessToken() {
  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function startLineLogin() {
  sessionStorage.setItem(RETURN_TO_KEY, window.location.pathname + window.location.search);
  window.location.assign(`${API_BASE_URL}/auth/line`);
}

export function clearAccessToken() {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
}

export { API_BASE_URL };
