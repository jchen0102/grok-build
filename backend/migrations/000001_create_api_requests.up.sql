CREATE TABLE api_requests (
    request_id  TEXT PRIMARY KEY,
    ts          TIMESTAMPTZ NOT NULL,
    service     TEXT NOT NULL,
    endpoint    TEXT NOT NULL,
    method      TEXT NOT NULL,
    status_code INTEGER NOT NULL CHECK (status_code >= 100 AND status_code <= 599),
    latency_ms  INTEGER NOT NULL CHECK (latency_ms >= 0)
);

CREATE INDEX api_requests_ts_idx ON api_requests (ts);
