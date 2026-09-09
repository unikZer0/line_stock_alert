import { EmptyState } from "../../../components/common/EmptyState";
import { ErrorState } from "../../../components/common/ErrorState";
import { LoadingState } from "../../../components/common/LoadingState";
import { PageLayout } from "../../../components/layout/PageLayout";
import { StockGrid } from "../components/StockGrid";
import { StockSearch } from "../components/StockSearch";
import { useStocks } from "../hooks/useStocks";

export function StocksPage() {
  const { query, setQuery, stocks, page, total, loading, error, load } = useStocks();

  return (
    <PageLayout>
      <section className="hero">
        <p className="eyebrow">US market</p>
        <h1>Find a stock</h1>
        <p>View its latest quote, then create an ABOVE or BELOW alert.</p>
      </section>
      <StockSearch query={query} onQueryChange={setQuery} onSearch={() => void load(1, false, query)} />
      {error && <ErrorState message={error} />}
      {loading && stocks.length === 0 && <LoadingState />}
      {!loading && !error && stocks.length === 0 && <EmptyState>No matching stocks were found.</EmptyState>}
      <StockGrid stocks={stocks} />
      {stocks.length < total && (
        <button className="load" disabled={loading} onClick={() => void load(page + 1, true, query)}>
          {loading ? "Loading…" : "Load more"}
        </button>
      )}
    </PageLayout>
  );
}
