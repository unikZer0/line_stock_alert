import type { FormEvent } from "react";

interface StockSearchProps {
  query: string;
  onQueryChange: (query: string) => void;
  onSearch: () => void;
}

export function StockSearch({ query, onQueryChange, onSearch }: StockSearchProps) {
  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    onSearch();
  };

  return (
    <form className="search" onSubmit={handleSubmit}>
      <input
        aria-label="Search US stocks"
        value={query}
        onChange={(event) => onQueryChange(event.target.value)}
        placeholder="Search AAPL, NVIDIA…"
      />
      <button type="submit">Search</button>
    </form>
  );
}
