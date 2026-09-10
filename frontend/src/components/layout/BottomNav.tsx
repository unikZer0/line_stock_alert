export function BottomNav() {
  const alerts = window.location.pathname.startsWith("/alerts");
  return <nav className="bottom-nav">
    <a className={!alerts ? "nav-active" : ""} href="/stocks"><span>⌁</span><small>MARKET</small></a>
    <a className={alerts ? "nav-active" : ""} href="/alerts"><span>♢</span><small>MY ALERTS</small></a>
  </nav>;
}
