import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { api, fmtNum, fmtRate, reasonLabel, type EndpointPage } from "../api";

const columns = [
  { key: "service", label: "服务" },
  { key: "endpoint", label: "接口" },
  { key: "method", label: "方法" },
  { key: "request_count", label: "请求数" },
  { key: "5xx_rate", label: "5xx 率" },
  { key: "avg_latency_ms", label: "平均延迟" },
  { key: "p95_latency_ms", label: "p95" },
  { key: "anomalous", label: "异常" },
] as const;

export default function Endpoints() {
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const [data, setData] = useState<EndpointPage | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const q = params.get("q") ?? "";
  const sort = params.get("sort") ?? "";
  const order = params.get("order") ?? "";
  const page = Number(params.get("page") || "1");
  const anomalous = params.get("anomalous") === "1";

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    api.endpoints({
      from: params.get("from"),
      to: params.get("to"),
      q,
      sort,
      order,
      page,
      per_page: 20,
      anomalous: anomalous ? "1" : undefined,
    })
      .then((res) => {
        if (!cancelled) setData(res);
      })
      .catch((err: Error) => {
        if (!cancelled) setError(err.message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [params.toString()]);

  const set = (key: string, value: string) => {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    if (key !== "page") next.set("page", "1");
    setParams(next);
  };

  const clickSort = (key: string) => {
    if (sort === key) set("order", order === "asc" ? "desc" : "asc");
    else {
      const next = new URLSearchParams(params);
      next.set("sort", key);
      next.set("order", key === "service" || key === "endpoint" || key === "method" ? "asc" : "desc");
      next.set("page", "1");
      setParams(next);
    }
  };

  return (
    <main className="page">
      <h1>接口聚合</h1>
      <div className="filters">
        <input
          placeholder="搜索 service / endpoint / method"
          defaultValue={q}
          onKeyDown={(e) => {
            if (e.key === "Enter") set("q", (e.target as HTMLInputElement).value);
          }}
        />
        <button className={anomalous ? "btn primary" : "btn"} onClick={() => set("anomalous", anomalous ? "" : "1")}>
          {anomalous ? "只看异常 · 开" : "只看异常"}
        </button>
      </div>
      {loading && <div className="banner">正在聚合…</div>}
      {error && <div className="banner error">{error}</div>}
      {data && data.total === 0 && <div className="banner empty">这个时间窗没有接口行。</div>}
      {data && data.total > 0 && (
        <section className="panel">
          <table>
            <thead>
              <tr>
                {columns.map((col) => (
                  <th key={col.key}>
                    <button onClick={() => clickSort(col.key)}>
                      {col.label}
                      {sort === col.key ? (order === "asc" ? " ↑" : " ↓") : ""}
                    </button>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.items.map((row) => (
                <tr
                  key={`${row.service}|${row.endpoint}|${row.method}`}
                  className={row.anomalous ? "bad" : ""}
                  style={{ cursor: "pointer" }}
                  onClick={() => {
                    const next = new URLSearchParams();
                    const from = params.get("from");
                    const to = params.get("to");
                    if (from) next.set("from", from);
                    if (to) next.set("to", to);
                    next.set("service", row.service);
                    next.set("endpoint", row.endpoint);
                    next.set("method", row.method);
                    navigate({ pathname: "/requests", search: `?${next.toString()}` });
                  }}
                >
                  <td>{row.service}</td>
                  <td>{row.endpoint}</td>
                  <td>{row.method}</td>
                  <td>{fmtNum(row.request_count)}</td>
                  <td>{fmtRate(row["5xx_rate"])}</td>
                  <td>{fmtNum(row.avg_latency_ms, 1)}</td>
                  <td>{fmtNum(row.p95_latency_ms)}</td>
                  <td>
                    {row.anomalous
                      ? row.reasons.map((reason) => (
                          <span className="chip" key={reason}>{reasonLabel(reason)}</span>
                        ))
                      : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="pager">
            <button disabled={page <= 1} onClick={() => set("page", String(page - 1))}>上一页</button>
            <span>
              {page} / {Math.max(1, Math.ceil(data.total / data.per_page))} · {data.total} 行
            </span>
            <button disabled={page * data.per_page >= data.total} onClick={() => set("page", String(page + 1))}>下一页</button>
          </div>
        </section>
      )}
    </main>
  );
}
