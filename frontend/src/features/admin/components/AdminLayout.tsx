import type { ReactNode } from "react";
import alertBotLogo from "../../../assets/alert-bot-logo.png";

const links = [
  ["/admin", "▥", "Dashboard & Metrics"], ["/admin/users", "♙", "Users"],
  ["/admin/alerts", "♢", "Alerts Engine"], ["/admin/stocks", "▤", "Stocks Directory"],
  ["/admin/line", "⌁", "LINE Push & Logs"], ["/admin/audit-logs", "✓", "Audit Logs"],
];

export function AdminLayout({ children }: { children: ReactNode }) {
  const path = window.location.pathname;
  return <div className="admin-shell"><aside className="admin-sidebar">
    <a className="admin-brand" href="/admin"><img src={alertBotLogo} alt="Alert Bot" /><b>ALERT BOT</b><em>ADMIN</em></a>
    <div className="admin-worker"><span><i /> WORKER</span><b>LIVE 15s</b></div>
    <small className="admin-nav-label">CORE NAVIGATION</small><nav>{links.map(([href, icon, label]) => <a href={href} key={href} className={path === href ? "active" : ""}><span>{icon}</span>{label}</a>)}</nav>
    <footer><span>◷ UTC+7 ICT</span><b>BANGKOK</b></footer>
  </aside><div className="admin-workspace"><header className="admin-topbar"><span>▣ NODE-TH01　/　<i /> DISPATCHER ONLINE</span><b>ADMIN CONSOLE</b></header><main>{children}</main></div></div>;
}
