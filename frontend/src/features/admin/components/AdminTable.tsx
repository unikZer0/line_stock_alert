import type { AdminRecord } from "../services/adminService";

function display(value: unknown) {
  if (value === null || value === undefined || value === "") return "—";
  if (typeof value === "boolean") return value ? "YES" : "NO";
  if (typeof value === "object") return JSON.stringify(value);
  if (typeof value === "string" && /^\d{4}-\d{2}-\d{2}T/.test(value)) return new Date(value).toLocaleString("en-GB", { timeZone: "Asia/Bangkok" });
  return String(value);
}

export function AdminTable({ rows, columns }: { rows: AdminRecord[]; columns: string[] }) {
  return <div className="admin-table-wrap"><table><thead><tr>{columns.map((column) => <th key={column}>{column.replaceAll("_", " ")}</th>)}</tr></thead>
    <tbody>{rows.map((row, index) => <tr key={String(row.id ?? row.symbol ?? index)}>{columns.map((column) => <td key={column} title={display(row[column])}><span className={column === "status" || column === "level" ? `admin-pill ${String(row[column]).toLowerCase()}` : ""}>{display(row[column])}</span></td>)}</tr>)}</tbody></table></div>;
}
