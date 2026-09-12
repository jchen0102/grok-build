# 2026-09-12 竖切记录

题目：C · API 调用质量监控台  
计划：十五轮已锁，栈锁 chi / pgx / golang-migrate / Vite / Recharts。  
约束：不重开产品轮；不实现「明确不做」；只测 p95 + 异常。

## 用户指令（摘要）

1. 按 `c-api-plan.md` 做完整竖切。
2. 只抄竞品/开源的形状：RED 总分、一个时间窗、avg+p95 并排、5xx≠4xx、导入后看批量 Insights。
3. 每做完一步立刻验证，不要堆到最后再测：
   1. migrate + 表
   2. 导入 + demo CSV
   3. `internal/metrics` + `go test`
   4. 总览 API + UI
   5. 聚合下钻
   6. 原始记录
   7. compose / README / ai-log

## 实现顺序与当时检查

| 步 | 做了什么 | 怎么证明绿 |
| --- | --- | --- |
| 1 | `api_requests` + golang-migrate embed | `\d api_requests`：PK=`request_id`，六列 NOT NULL，`ts` 索引，状态码/延迟 check |
| 2 | `POST /api/imports` + `data/demo.csv` | demo 7591 成功；再导 7591 库内重复（样本封顶 5）；缺文件/缺表头 400；混合行分类正确 |
| 3 | nearest-rank p95 + 异常分类 | `go test ./internal/metrics`：n=0/1/19/20、乱序、单/双原因、4xx 不算错误、计数=行数 |
| 4 | `GET /api/overview` + 看板 | 六指标与库手算一致；Playwright：7591 / 4.47% / 162.6 / 438 / 5 / 3 + 小时图 |
| 5 | `GET /api/endpoints` + 表 | 默认异常优先；`anomalous=1` 3 行；cart n=15 不标异常；点异常卡带同一 UTC 窗 |
| 6 | `GET /api/requests` | 过滤数=库计数；`ts DESC, request_id DESC`；点聚合行进原始记录并可翻页 |
| 7 | compose + 中文 README + 本目录 | `docker compose up --build` 后导入 7591 行；nginx 反代总览数字与本机手算一致；看板→异常表→原始记录可点透 |

## 口径（未改计划，只落代码）

- p95：`ceil(0.95*n)` 1-based 夹到 n
- 异常：n≥20 且（5xx 率≥10% 或 p95≥1000ms），双原因都展示
- 时间：UTC，闭区间，小时桶左闭右开
- 导入失败样本封顶：5

## 未做（按计划）

登录 / 异步导入 / JSON / 可配阈值 / 详情页 / OTLP / Prometheus / Grafana / 告警 / 导入与 HTTP 的自动化测试 / 前端 E2E 作为提交物。Playwright 只用于本机验收，不进仓库。
