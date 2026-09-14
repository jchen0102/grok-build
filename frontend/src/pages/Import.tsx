import { useState } from "react";
import { api, type ImportResult } from "../api";

const labels: Record<string, string> = {
  missing_column: "缺列 / 空值",
  invalid_time: "时间非法",
  invalid_status: "状态码非法",
  invalid_latency: "延迟非法",
  duplicate_in_file: "文件内重复",
  duplicate_in_db: "库内重复",
};

export default function ImportPage() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [result, setResult] = useState<ImportResult | null>(null);

  const onFile = async (file?: File) => {
    if (!file) return;
    setBusy(true);
    setError("");
    try {
      setResult(await api.importFile(file));
    } catch (err) {
      setError(err instanceof Error ? err.message : "导入失败");
    } finally {
      setBusy(false);
    }
  };

  return (
    <main className="page">
      <h1>导入调用日志</h1>
      <p style={{ color: "#b7a48c" }}>同步解析 CSV。有效行尽量入库。时间窗不参与导入。</p>
      <div className="drop">
        <p>选择 `data/demo.csv` 或任意七列表。</p>
        <input type="file" accept=".csv,text/csv" onChange={(e) => onFile(e.target.files?.[0])} />
      </div>
      {busy && <div className="banner">正在导入…</div>}
      {error && <div className="banner error">{error}</div>}
      {result && (
        <>
          <div className="insight">
            <article className="card moss"><div className="kicker">成功</div><div className="value">{result.summary.success}</div></article>
            <article className="card amber"><div className="kicker">重复</div><div className="value">{result.summary.duplicate}</div></article>
            <article className="card terra"><div className="kicker">失败</div><div className="value">{result.summary.failed}</div></article>
          </div>
          {Object.entries(result.failures).map(([key, bucket]) => (
            <section className="panel" key={key} style={{ marginBottom: 12 }}>
              <div className="panel-head">
                <h2>{labels[key] ?? key}</h2>
                <span>{bucket.count}</span>
              </div>
              {bucket.samples.length === 0 ? (
                <p style={{ color: "#b7a48c" }}>无样本</p>
              ) : (
                <pre>{JSON.stringify(bucket.samples, null, 2)}</pre>
              )}
            </section>
          ))}
        </>
      )}
    </main>
  );
}
