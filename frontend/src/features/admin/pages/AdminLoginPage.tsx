import { useState, type FormEvent } from "react";
import alertBotLogo from "../../../assets/alert-bot-logo.png";
import { saveSession } from "../../auth/services/sessionService";
import { adminLogin } from "../services/adminService";

export function AdminLoginPage() {
  const [email, setEmail] = useState(""), [password, setPassword] = useState(""), [error, setError] = useState(""), [loading, setLoading] = useState(false);
  const submit = async (event: FormEvent) => { event.preventDefault(); setLoading(true); setError(""); try { const tokens = await adminLogin(email, password); saveSession(tokens.access_token, tokens.refresh_token); window.location.assign("/admin"); } catch (cause) { setError(cause instanceof Error ? cause.message : "Administrator login failed."); } finally { setLoading(false); } };
  return <main className="admin-login"><form onSubmit={(event) => void submit(event)}><img src={alertBotLogo} alt="Alert Bot" /><span>ALERT BOT / ADMIN</span><h1>Console sign in</h1><p>Use an active administrator account.</p><label>Email<input type="email" required value={email} onChange={(event) => setEmail(event.target.value)} /></label><label>Password<input type="password" required value={password} onChange={(event) => setPassword(event.target.value)} /></label><button disabled={loading}>{loading ? "Signing in…" : "Sign in"}</button>{error && <p className="error" role="alert">{error}</p>}</form></main>;
}
