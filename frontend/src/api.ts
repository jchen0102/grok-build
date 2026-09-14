export type ApiError = { error: { code: string; message: string } };

export type Overview = {
  from: string | null;
  to: string | null;
  empty: boolean;
  metrics: {
    request_count: number;
    error_5xx_rate: number | null;
    avg_latency_ms: number | null;
    p95_latency_ms: number | null;
    service_count: number;
    anomalous_endpoint_count: number;
  };
  trend: Array<{
    hour: string;
    requests: number;
    "5xx_count": number;
    "5xx_rate": number | null;
  }>;
};

export type ImportResult = {
  summary: { success: number; duplicate: number; failed: number };
  failures: Record<string, { count: number; samples: Array<{ line: number; request_id?: string; detail: string }> }>;
};

export type EndpointRow = {
  service: string;
  endpoint: string;
  method: string;
  request_count: number;
  "5xx_count": number;
  "5xx_rate": number | null;
  avg_latency_ms: number | null;
  p95_latency_ms: number | null;
  anomalous: boolean;
  reasons: string[];
};

export type EndpointPage = {
  from: string | null;
  to: string | null;
  page: number;
  per_page: number;
  total: number;
  items: EndpointRow[];
};

export type RequestRow = {
  request_id: string;
  ts: string;
  service: string;
  endpoint: string;
  method: string;
  status_code: number;
  latency_ms: number;
};

export type RequestPage = {
  from: string | null;
  to: string | null;
  page: number;
  per_page: number;
  total: number;
  items: RequestRow[];
};

async function read<T>(res: Response): Promise<T> {
  const body = await res.json();
  if (!res.ok) {
    const err = body as ApiError;
    throw new Error(err.error?.message || `HTTP ${res.status}`);
  }
  return body as T;
}

export function qs(params: Record<string, string | number | undefined | null>): string {
  const usp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null || v === "") continue;
    usp.set(k, String(v));
  }
  const s = usp.toString();
  return s ? `?${s}` : "";
}

export const api = {
  overview: (from?: string, to?: string) =>
    fetch(`/api/overview${qs({ from, to })}`).then(read<Overview>),
  importFile: async (file: File) => {
    const form = new FormData();
    form.set("file", file);
    return fetch("/api/imports", { method: "POST", body: form }).then(read<ImportResult>);
  },
  endpoints: (params: Record<string, string | number | undefined | null>) =>
    fetch(`/api/endpoints${qs(params)}`).then(read<EndpointPage>),
  requests: (params: Record<string, string | number | undefined | null>) =>
    fetch(`/api/requests${qs(params)}`).then(read<RequestPage>),
};

export function toInput(iso?: string | null): string {
  if (!iso) return "";
  return iso.slice(0, 16);
}

export function fromInput(value: string): string {
  if (!value) return "";
  return value.length === 16 ? `${value}:00Z` : `${value}Z`;
}

export function fmtNum(v: number | null | undefined, digits = 0): string {
  if (v === null || v === undefined || Number.isNaN(v)) return "—";
  return v.toLocaleString("en-US", { maximumFractionDigits: digits, minimumFractionDigits: digits });
}

export function fmtRate(v: number | null | undefined): string {
  if (v === null || v === undefined) return "—";
  return `${(v * 100).toFixed(2)}%`;
}

export function reasonLabel(code: string): string {
  if (code === "high_5xx_rate") return "5xx 率 ≥ 10%";
  if (code === "high_p95") return "p95 ≥ 1000ms";
  return code;
}
