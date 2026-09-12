import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { api, fmtNum, type RequestPage } from "../api";

export default function Requests() {
  const [params, setParams] = useSearchParams();
  const [data, setData] = useState<RequestPage | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const page = Number(params.get("page") || "1");

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    api.requests({
      from: params.get("from"),
      to: params.get("to"),
      service: params.get("service"),
      endpoint: params.get("endpoint"),
      method: params.get("method"),
      status_code: params.get("status_code"),
      page,
      per_page: 20,
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

  return (
    <main className="page">
      <h1>原始记录</h1>
      <div className="filters">
        <input defaultValue={params.get("service") ?? ""} placeholder="service" onBlur={(e) => set("service", e.target.value)} />
        <input defaultValue={params.get("endpoint") ?? ""} placeholder="endpoint" onBlur={(e) => set("endpoint", e.target.value)} />
        <input defaultValue={params.get("method") ?? ""} placeholder="method" onBlur={(e) => set("method", e.target.value)} />
        <input defaultValue={params.get("status_code") ?? ""} placeholder="status_code" onBlur={(e) => set("status_code", e.target.value)} />
      </div>
      {loading && <div className="banner">正在读取记录…</div>}
      {error && <div className="banner error">{error}</div>}
      {data && data.total === 0 && <div className="banner empty">这个筛选没有原始行。</div>}
      {data && data.total > 0 && (
        <section className="panel">
          <table>
            <thead>
              <tr>
                <th>时间 (UTC)</th>
                <th>request_id</th>
                <th>服务</th>
                <th>接口</th>
                <th>方法</th>
                <th>状态码</th>
                <th>延迟 ms</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((row) => (
                <tr key={row.request_id} className={row.status_code >= 500 ? "bad" : ""}>
                  <td>{row.ts.replace("T", " ").replace("Z", "")}</td>
                  <td>{row.request_id}</td>
                  <td>{row.service}</td>
                  <td>{row.endpoint}</td>
                  <td>{row.method}</td>
                  <td>{row.status_code}</td>
                  <td>{fmtNum(row.latency_ms)}</td>
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
