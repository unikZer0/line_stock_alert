import { useEffect, useState } from "react";
import { getMarketStatus } from "../services/stockService";
import type { MarketSession } from "../types/stock";
import { MarketStatus } from "./MarketStatus";

export function GlobalMarketStatus() {
  const [session, setSession] = useState<MarketSession | null>(null);
  useEffect(() => {
    let active = true;
    const load = () => getMarketStatus().then((response) => { if (active) setSession(response.data); }).catch(() => undefined);
    void load();
    const timer = window.setInterval(load, 60_000);
    return () => { active = false; window.clearInterval(timer); };
  }, []);
  if (!session) return null;
  return <div className="global-market-status"><MarketStatus status={session.status} until={session.until} /></div>;
}
