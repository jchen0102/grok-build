package metrics

import (
	"math"
	"sort"
)

const (
	ReasonHigh5xx = "high_5xx_rate"
	ReasonHighP95 = "high_p95"
	AnomalyMinN   = 20
	Anomaly5xx    = 0.10
	AnomalyP95MS  = 1000
)

type Anomaly struct {
	Anomalous bool     `json:"anomalous"`
	Reasons   []string `json:"reasons"`
}

func Is5xx(status int) bool {
	return status >= 500 && status <= 599
}

func P95(latencies []int) *int {
	n := len(latencies)
	if n == 0 {
		return nil
	}
	sorted := append([]int(nil), latencies...)
	sort.Ints(sorted)
	rank := int(math.Ceil(0.95 * float64(n)))
	if rank < 1 {
		rank = 1
	}
	if rank > n {
		rank = n
	}
	v := sorted[rank-1]
	return &v
}

func FiveXXRate(statuses []int) *float64 {
	n := len(statuses)
	if n == 0 {
		return nil
	}
	var n5 int
	for _, status := range statuses {
		if Is5xx(status) {
			n5++
		}
	}
	rate := float64(n5) / float64(n)
	return &rate
}

func ClassifyAnomaly(n int, fiveXXRate *float64, p95 *int) Anomaly {
	out := Anomaly{Reasons: []string{}}
	if n < AnomalyMinN {
		return out
	}
	if fiveXXRate != nil && *fiveXXRate >= Anomaly5xx {
		out.Anomalous = true
		out.Reasons = append(out.Reasons, ReasonHigh5xx)
	}
	if p95 != nil && *p95 >= AnomalyP95MS {
		out.Anomalous = true
		out.Reasons = append(out.Reasons, ReasonHighP95)
	}
	return out
}

type EndpointKey struct {
	Service  string
	Endpoint string
	Method   string
}

type EndpointStat struct {
	Key          EndpointKey
	RequestCount int
	FiveXXCount  int
	FiveXXRate   *float64
	AvgLatencyMS *float64
	P95LatencyMS *int
	Anomaly
}

func SummarizeEndpoint(key EndpointKey, statuses []int, latencies []int) EndpointStat {
	n := len(latencies)
	if len(statuses) < n {
		n = len(statuses)
	}
	stat := EndpointStat{
		Key:          key,
		RequestCount: n,
		FiveXXRate:   FiveXXRate(statuses[:n]),
		P95LatencyMS: P95(latencies[:n]),
	}
	for _, status := range statuses[:n] {
		if Is5xx(status) {
			stat.FiveXXCount++
		}
	}
	if n > 0 {
		var sum int
		for _, lat := range latencies[:n] {
			sum += lat
		}
		avg := float64(sum) / float64(n)
		stat.AvgLatencyMS = &avg
	}
	stat.Anomaly = ClassifyAnomaly(stat.RequestCount, stat.FiveXXRate, stat.P95LatencyMS)
	return stat
}
