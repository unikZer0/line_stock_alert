import { EmptyState } from "../../../components/common/EmptyState";
import { ErrorState } from "../../../components/common/ErrorState";
import { LoadingState } from "../../../components/common/LoadingState";
import { PageLayout } from "../../../components/layout/PageLayout";
import { StockGrid } from "../components/StockGrid";
import { GlobalMarketStatus } from "../components/GlobalMarketStatus";
import { StockSearch } from "../components/StockSearch";
import { useStocks } from "../hooks/useStocks";

export function StocksPage() {
  const { query, setQuery, stocks, page, total, loading, error, load } = useStocks();
  return <PageLayout>
    <div className="telemetry"><span><i /> US EQUITIES FEED</span><b>FINNHUB · LIVE</b></div>
    <GlobalMarketStatus />
    <section className="page-title"><div><h1>Market</h1><p>DISCOVER & MONITOR US STOCKS</p></div><span className="market-count">{total.toLocaleString()} SYMBOLS</span></section>
    <StockSearch query={query} suggestions={stocks} loading={loading} onQueryChange={setQuery} onSearch={() => void load(1, false, query)} />
    {error && <ErrorState message={error} />}{loading && stocks.length === 0 && <LoadingState />}
    {!loading && !error && stocks.length === 0 && <EmptyState>No matching stocks were found.</EmptyState>}
    <StockGrid stocks={stocks} />
    {stocks.length < total && <button className="load" disabled={loading} onClick={() => void load(page + 1, true, query)}>{loading ? "Loading…" : "Load more"}</button>}
  </PageLayout>;
}
