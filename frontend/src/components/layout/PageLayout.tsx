import type { ReactNode } from "react";

interface PageLayoutProps {
  children: ReactNode;
}

export function PageLayout({ children }: PageLayoutProps) {
  return (
    <>
      <header>
        <a className="brand" href="/stocks">Stock Alert</a>
        <nav>
          <a href="/stocks">Stocks</a>
          <a href="/alerts">My Alerts</a>
        </nav>
      </header>
      <main>{children}</main>
    </>
  );
}
