import React from "react";
import ReactDOM from "react-dom/client";
import "./styles.css";

function App() {
  return (
    <main className="page">
      <section className="card">
        <p className="eyebrow">Stock Alert</p>
        <h1>Your market alerts, in one place.</h1>
        <p>The React frontend is running and ready for the first feature.</p>
      </section>
    </main>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
