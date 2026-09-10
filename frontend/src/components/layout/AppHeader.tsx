import alertBotLogo from "../../assets/alert-bot-logo.png";

export function AppHeader() {
  return <header className="app-header"><div className="header-inner">
    <a className="brand-lockup" href="/stocks"><img className="brand-logo" src={alertBotLogo} alt="Alert Bot" /><span><b>ALERT BOT</b><small><i /> LINE CONNECTED</small></span></a>
    <span className="profile-icon" aria-label="Profile">●</span>
  </div></header>;
}
