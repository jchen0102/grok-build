package compute

import (
	"time"

	"github.com/jchen0102/grok-build/internal/metrics"
	"github.com/jchen0102/grok-build/internal/store"
)

type Overview struct {
	From    *time.Time      `json:"from"`
	To      *time.Time      `json:"to"`
	Empty   bool            `json:"empty"`
	Metrics OverviewMetrics `json:"metrics"`
	Trend   []HourBucket    `json:"trend"`
}

type OverviewMetrics struct {
	RequestCount           int      `json:"request_count"`
	Error5xxRate           *float64 `json:"error_5xx_rate"`
	AvgLatencyMS           *float64 `json:"avg_latency_ms"`
	P95LatencyMS           *int     `json:"p95_latency_ms"`
	ServiceCount           int      `json:"service_count"`
	AnomalousEndpointCount int      `json:"anomalous_endpoint_count"`
}

type HourBucket struct {
	Hour        time.Time `json:"hour"`
	Requests    int       `json:"requests"`
	FiveXXCount int       `json:"5xx_count"`
	FiveXXRate  *float64  `json:"5xx_rate"`
}

type EndpointRow struct {
	Service      string   `json:"service"`
	Endpoint     string   `json:"endpoint"`
	Method       string   `json:"method"`
	RequestCount int      `json:"request_count"`
	FiveXXCount  int      `json:"5xx_count"`
	FiveXXRate   *float64 `json:"5xx_rate"`
	AvgLatencyMS *float64 `json:"avg_latency_ms"`
	P95LatencyMS *int     `json:"p95_latency_ms"`
	Anomalous    bool     `json:"anomalous"`
	Reasons      []string `json:"reasons"`
}

func BuildOverview(rows []store.Request, from, to time.Time) Overview {
	out := Overview{
		From:  &from,
		To:    &to,
		Empty: len(rows) == 0,
		Trend: hourTrend(rows, from, to),
	}
	if len(rows) == 0 {
		return out
	}

	statuses := make([]int, len(rows))
	latencies := make([]int, len(rows))
	services := map[string]struct{}{}
	for i, row := range rows {
		statuses[i] = row.StatusCode
		latencies[i] = row.LatencyMS
		services[row.Service] = struct{}{}
	}

	var sum int
	for _, lat := range latencies {
		sum += lat
	}
	avg := float64(sum) / float64(len(latencies))

	out.Metrics = OverviewMetrics{
		RequestCount:           len(rows),
		Error5xxRate:           metrics.FiveXXRate(statuses),
		AvgLatencyMS:           &avg,
		P95LatencyMS:           metrics.P95(latencies),
		ServiceCount:           len(services),
		AnomalousEndpointCount: countAnomalous(AggregateEndpoints(rows)),
	}
	return out
}

func AggregateEndpoints(rows []store.Request) []EndpointRow {
	type acc struct {
		statuses  []int
		latencies []int
	}
	grouped := map[metrics.EndpointKey]*acc{}
	for _, row := range rows {
		key := metrics.EndpointKey{Service: row.Service, Endpoint: row.Endpoint, Method: row.Method}
		bucket, ok := grouped[key]
		if !ok {
			bucket = &acc{}
			grouped[key] = bucket
		}
		bucket.statuses = append(bucket.statuses, row.StatusCode)
		bucket.latencies = append(bucket.latencies, row.LatencyMS)
	}

	out := make([]EndpointRow, 0, len(grouped))
	for key, bucket := range grouped {
		stat := metrics.SummarizeEndpoint(key, bucket.statuses, bucket.latencies)
		out = append(out, EndpointRow{
			Service:      key.Service,
			Endpoint:     key.Endpoint,
			Method:       key.Method,
			RequestCount: stat.RequestCount,
			FiveXXCount:  stat.FiveXXCount,
			FiveXXRate:   stat.FiveXXRate,
			AvgLatencyMS: stat.AvgLatencyMS,
			P95LatencyMS: stat.P95LatencyMS,
			Anomalous:    stat.Anomalous,
			Reasons:      stat.Reasons,
		})
	}
	return out
}

func countAnomalous(rows []EndpointRow) int {
	var n int
	for _, row := range rows {
		if row.Anomalous {
			n++
		}
	}
	return n
}

func hourTrend(rows []store.Request, from, to time.Time) []HourBucket {
	start := from.UTC().Truncate(time.Hour)
	end := to.UTC().Truncate(time.Hour)
	if end.Before(start) {
		return []HourBucket{}
	}

	counts := map[time.Time][2]int{}
	for _, row := range rows {
		h := row.TS.UTC().Truncate(time.Hour)
		pair := counts[h]
		pair[0]++
		if metrics.Is5xx(row.StatusCode) {
			pair[1]++
		}
		counts[h] = pair
	}

	var out []HourBucket
	for h := start; !h.After(end); h = h.Add(time.Hour) {
		pair := counts[h]
		bucket := HourBucket{Hour: h, Requests: pair[0], FiveXXCount: pair[1]}
		if pair[0] > 0 {
			rate := float64(pair[1]) / float64(pair[0])
			bucket.FiveXXRate = &rate
		}
		out = append(out, bucket)
	}
	return out
}
