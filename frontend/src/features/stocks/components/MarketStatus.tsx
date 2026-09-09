import { useEffect, useState } from "react";

const localTime = new Intl.DateTimeFormat("en-GB", {
  timeZone: "Asia/Bangkok", weekday: "short", hour: "2-digit", minute: "2-digit", hour12: false,
});

function countdown(until: Date, now: number) {
  const totalMinutes = Math.max(0, Math.ceil((until.getTime() - now) / 60_000));
  const days = Math.floor(totalMinutes / 1440);
  const hours = Math.floor((totalMinutes % 1440) / 60);
  const minutes = totalMinutes % 60;
  return [days ? `${days}d` : "", hours ? `${hours}h` : "", `${minutes}m`].filter(Boolean).join(" ");
}

export function MarketStatus({ status, until }: { status: string; until: string }) {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 30_000);
    return () => window.clearInterval(timer);
  }, []);
  const transition = new Date(until);
  const open = status === "OPEN";
  return <div className={`market-status ${open ? "market-open" : "market-closed"}`}>
    <span className="market-dot" aria-hidden="true" />
    <span><small>Market</small><strong>{open ? "Open" : "Closed"}</strong></span>
    <span><small>{open ? "Open until" : "Closed until"}</small><strong>{localTime.format(transition)} น.</strong></span>
    <span><small>{open ? "Closes in" : "Opens in"}</small><strong>{countdown(transition, now)}</strong></span>
  </div>;
}
