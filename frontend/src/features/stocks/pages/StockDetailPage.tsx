import { useState } from "react";

import { ErrorState } from "../../../components/common/ErrorState";
import { LoadingState } from "../../../components/common/LoadingState";
import { PageLayout } from "../../../components/layout/PageLayout";
import { AlertForm } from "../../alerts/components/AlertForm";
import { SymbolAlerts } from "../../alerts/components/SymbolAlerts";
import { StockHistoryChart } from "../components/StockHistoryChart";
import { StockQuotePanel } from "../components/StockQuotePanel";
import { useStockQuote } from "../hooks/useStockQuote";

interface StockDetailPageProps {
  symbol: string;
}

export function StockDetailPage({ symbol }: StockDetailPageProps) {
  const { quote, loading, error } = useStockQuote(symbol);
  const [alertsRefreshKey, setAlertsRefreshKey] = useState(0);

  return (
    <PageLayout>
      <a className="back" href="/stocks">← All stocks</a>
      {loading && <LoadingState />}
      {error && <ErrorState message={error} />}
      {quote && <StockQuotePanel quote={quote} />}
      <StockHistoryChart symbol={symbol} />
      {quote && (
        <>
          <AlertForm symbol={quote.symbol} initialPrice={quote.price} onCreated={() => setAlertsRefreshKey((value) => value + 1)} />
          <SymbolAlerts symbol={quote.symbol} refreshKey={alertsRefreshKey} />
        </>
      )}
    </PageLayout>
  );
}
