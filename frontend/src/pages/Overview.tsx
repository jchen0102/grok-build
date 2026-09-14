import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Bar, CartesianGrid, ComposedChart, Legend, Line, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { api, fmtNum, fmtRate, type Overview } from "../api";

export default function OverviewPage() {
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const from = params.get("from") ?? undefined;
  const to = params.get("to") ?? undefined;
  const [data, setData] = useState<Overview | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [mode, setMode] = useState<"count" | "rate">("count");

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    api.overview(from, to)
      .then((res) => {
        if (cancelled) return;
        setData(res);
        if (!from && !to && res.from && res.to) {
          const next = new URLSearchParams(params);
          next.set("from", res.from);
          next.set("to", res.to);
          setParams(next, { replace: true });
        }
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
  }, [from, to]);

  const chart = useMemo(
    () =>
      (data?.trend ?? []).map((b) => ({
        hour: b.hour.slice(5, 16).replace("T", " "),
        requests: b.requests,
        five: b["5xx_count"],
        rate: b["5xx_rate"] == null ? null : Number((b["5xx_rate"] * 100).toFixed(2)),
      })),
    [data],
  );

  const m = data?.metrics;

  return (
    <main className="page">
      {loading && <div className="banner">正在计算当前时间窗…</div>}
      {error && <div className="banner error">{error}</div>}
      {data?.empty && (
        <div className="banner empty">
          库是空的，看板壳留在这里。{" "}
          <a href="/import" onClick={(e) => { e.preventDefault(); navigate("/import"); }}>去导入 demo.csv</a>
        </div>
      )}

      <div className="cards">
        <article className="card">
          <div className="kicker">请求数</div>
          <div className="value">{fmtNum(m?.request_count)}</div>
        </article>
        <article className="card terra">
          <div className="kicker">5xx 率</div>
          <div className="value">{fmtRate(m?.error_5xx_rate)}</div>
        </article>
        <article className="card">
          <div className="kicker">平均延迟</div>
          <div className="value">{fmtNum(m?.avg_latency_ms, 1)}</div>
        </article>
        <article className="card amber">
          <div className="kicker">p95 延迟</div>
          <div className="value">{fmtNum(m?.p95_latency_ms)}</div>
        </article>
        <article className="card moss">
          <div className="kicker">服务数</div>
          <div className="value">{fmtNum(m?.service_count)}</div>
        </article>
        <article
          className="card terra clickable"
          onClick={() => {
            const next = new URLSearchParams(params);
            next.set("anomalous", "1");
            navigate({ pathname: "/endpoints", search: `?${next.toString()}` });
          }}
        >
          <div className="kicker">异常接口数</div>
          <div className="value">{fmtNum(m?.anomalous_endpoint_count)}</div>
        </article>
      </div>

      <section className="panel">
        <div className="panel-head">
          <h2>小时趋势</h2>
          <div className="toggle">
            <button className={mode === "count" ? "on" : ""} onClick={() => setMode("count")}>请求量 + 5xx 次数</button>
            <button className={mode === "rate" ? "on" : ""} onClick={() => setMode("rate")}>5xx 率</button>
          </div>
        </div>
        <div style={{ height: 320 }}>
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart data={chart}>
              <CartesianGrid stroke="#3a2f24" vertical={false} />
              <XAxis dataKey="hour" stroke="#b7a48c" tick={{ fontSize: 11 }} />
              <YAxis yAxisId="left" stroke="#f3e6d2" tick={{ fontSize: 11 }} />
              {mode === "count" ? (
                <YAxis yAxisId="right" orientation="right" stroke="#d4543a" tick={{ fontSize: 11 }} />
              ) : (
                <YAxis yAxisId="right" orientation="right" stroke="#d4543a" tick={{ fontSize: 11 }} unit="%" />
              )}
              <Tooltip contentStyle={{ background: "#16110d", border: "1px solid #3a2f24" }} />
              <Legend />
              <Bar yAxisId="left" dataKey="requests" name="请求量" fill="#7eb3c9" />
              {mode === "count" ? (
                <Line yAxisId="right" type="monotone" dataKey="five" name="5xx 次数" stroke="#d4543a" dot={false} />
              ) : (
                <Line yAxisId="right" type="monotone" dataKey="rate" name="5xx 率 %" stroke="#e4a34a" dot={false} connectNulls={false} />
              )}
            </ComposedChart>
          </ResponsiveContainer>
        </div>
      </section>
    </main>
  );
}
