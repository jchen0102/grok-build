# 调用质量台（题目 C）

只读质量看板：导入 CSV 调用日志，看总分、小时趋势、异常接口、原始记录。无登录。默认进总览。空库停在看板壳，用 CTA 去导入。

```bash
docker compose up --build
```

浏览器打开 `http://localhost:5173`，到「导入」上传 [`data/demo.csv`](data/demo.csv)。同一 UTC 时间窗贯穿看板 / 聚合 / 原始记录。

栈（已锁定，不再换）：**chi + pgx + golang-migrate + Vite + Recharts**。

AI 对话与实现记录见 [`ai-log/`](ai-log/)，不要在本 README 里粘贴对话。

---

## 怎么跑

1. `docker compose up --build`
2. UI：`http://localhost:5173`（nginx 反代 `/api`）
3. API：`http://localhost:8080`
4. 导入 `data/demo.csv`（约 7.6k 行，含 5xx-only / p95-only / 双原因 / n&lt;20 对照）

本地开发（可选）：Postgres 起好后

```bash
export DATABASE_URL=postgres://api:api@127.0.0.1:5432/api_quality?sslmode=disable
cd backend && go run ./cmd/server
cd frontend && npm install && npm run dev
```

自动化测试只覆盖口径：

```bash
cd backend && go test ./internal/metrics
```

---

## 导入策略

- 同步 `POST /api/imports`，字段名 `file`。
- 流式 `encoding/csv` 解析，500 行一批 `INSERT ... ON CONFLICT DO NOTHING`。
- 有效行尽量入库。整文件不可用（缺文件 / 缺表头）返回 **400**；部分成功返回 **200 + summary**。
- 计数：`success` / `duplicate` / `failed`。
- 失败分类：缺列、时间非法、状态码非法、延迟非法、文件内重复、库内重复。每类样本最多 5 条。
- 空 `request_id` 记失败，不入库（Postgres UNIQUE 允许多个 NULL，空不能当 NULL）。
- 只 `trim`，不去折叠大小写。
- 七列：`request_id,timestamp,service,endpoint,method,status_code,latency_ms`。

---

## 去重

- `request_id` 全局唯一，主键，**先到先得**。
- 文件内先用 map 去重，再插入；后到的记 `duplicate_in_file`。
- 库内冲突记 `duplicate_in_db`。`duplicate = 文件内 + 库内`。

---

## p95

- 在 `internal/metrics` 对**过滤后的同一份行**做精确分位，不用直方图，也不对小时 p95 再平均。
- 算法：nearest-rank，`rank = ceil(0.95 * n)`，1-based，越界夹到 n。
- `n=0` → `null`；`n=1` → 该值。
- 样本 = 时间窗内全部入库状态码（2xx/3xx/4xx/5xx）。4xx 计入分母和延迟，不计入 5xx 率。

---

## 时区

- 库列 `timestamptz`，CSV 与查询都按 **UTC**。
- 默认窗 = 库内 `min(ts)`–`max(ts)`。查询闭区间 `[from, to]`。
- 小时桶 `[h, h+1)`。空桶：次数 0，比率空。
- UI 时间控件按 UTC 墙钟读写，不转浏览器本地时区。

---

## 性能

- 当前量（演示约 1 万行量级）：Postgres 按时间窗滤行，Go 内存算总览 / 趋势 / 聚合 / p95 / 异常。
- 唯一额外索引：`INDEX (ts)`。原始列表 `ORDER BY ts DESC, request_id DESC` + OFFSET。
- 导入禁止逐行 HTTP；一批 500 条写入。
- 1 万行闭区间全扫进内存，先按题面做。慢了再加 `INDEX(ts)` 之外的手段。

---

## ×1000 演进（只写在这里，代码按当前量）

数据到百万 / 千万时，不要继续「整窗拉进 Go」：

1. 保留 `ts` 索引，列表改 keyset（`ts, request_id`）替代深 OFFSET。
2. 导入改 `COPY` / 分区表按日或按周切。
3. p95 下推 SQL（`percentile_disc` / 有界 sketch），或按 `service+endpoint+method+hour` 预聚合后再算。
4. 再往后才是物化视图 / rollup，而不是现在就加批次表和预聚合表。

---

## 异常规则（写死）

接口键：`service + endpoint + method`。

`n ≥ 20` 且（**5xx 率 ≥ 10%** 或 **p95 ≥ 1000ms**）。两项都中则两个原因都展示。`n < 20` 仍列出，不标异常。总览「异常接口数」= 时间窗内异常聚合行，不受搜索影响。5xx 只计 500–599；**4xx 不算错误率**。

---

## HTTP

| 方法 | 路径 | 作用 |
| --- | --- | --- |
| `POST` | `/api/imports` | CSV 导入 |
| `GET` | `/api/overview` | 六指标 + 小时趋势 |
| `GET` | `/api/endpoints` | 聚合表 |
| `GET` | `/api/requests` | 原始记录 |

时间窗：`from` / `to`，UTC RFC3339，闭区间。分页：`page` + `per_page`，默认 20，上限 100。错误体：`{ "error": { "code", "message" } }`。

---

## 明确不做

登录、角色、审计、异步导入、JSON 摄入、可配阈值、详情页、实时摄入、Watchdog / ML、OTLP、Prometheus、Grafana、告警、把 4xx 计入错误率。

---

## 目录

```
backend/     Go API + golang-migrate
frontend/    Vite + React + Recharts
data/        demo.csv
ai-log/      AI 记录（README 只放指针）
```
