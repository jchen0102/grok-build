package compute

import (
	"sort"
	"strings"
)

type EndpointQuery struct {
	Search    string
	Anomalous bool
	Sort      string
	Order     string
	Page      int
	PerPage   int
}

type EndpointPage struct {
	From    *string       `json:"from"`
	To      *string       `json:"to"`
	Page    int           `json:"page"`
	PerPage int           `json:"per_page"`
	Total   int           `json:"total"`
	Items   []EndpointRow `json:"items"`
}

func PageEndpoints(rows []EndpointRow, q EndpointQuery) EndpointPage {
	filtered := make([]EndpointRow, 0, len(rows))
	needle := strings.ToLower(strings.TrimSpace(q.Search))
	for _, row := range rows {
		if q.Anomalous && !row.Anomalous {
			continue
		}
		if needle != "" {
			hay := strings.ToLower(row.Service + " " + row.Endpoint + " " + row.Method)
			if !strings.Contains(hay, needle) {
				continue
			}
		}
		if row.Reasons == nil {
			row.Reasons = []string{}
		}
		filtered = append(filtered, row)
	}

	less := endpointLess(q.Sort, strings.ToLower(q.Order))
	sort.SliceStable(filtered, func(i, j int) bool {
		return less(filtered[i], filtered[j])
	})

	page := q.Page
	if page < 1 {
		page = 1
	}
	per := q.PerPage
	if per < 1 {
		per = 20
	}
	if per > 100 {
		per = 100
	}
	start := (page - 1) * per
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + per
	if end > len(filtered) {
		end = len(filtered)
	}

	items := filtered[start:end]
	if items == nil {
		items = []EndpointRow{}
	}
	return EndpointPage{
		Page:    page,
		PerPage: per,
		Total:   len(filtered),
		Items:   items,
	}
}

func endpointLess(sortKey, order string) func(a, b EndpointRow) bool {
	desc := order != "asc"
	val := func(row EndpointRow) float64 {
		switch sortKey {
		case "service":
			return 0
		case "endpoint":
			return 0
		case "method":
			return 0
		case "5xx_rate":
			if row.FiveXXRate == nil {
				return -1
			}
			return *row.FiveXXRate
		case "avg_latency_ms":
			if row.AvgLatencyMS == nil {
				return -1
			}
			return *row.AvgLatencyMS
		case "p95_latency_ms":
			if row.P95LatencyMS == nil {
				return -1
			}
			return float64(*row.P95LatencyMS)
		case "anomalous":
			if row.Anomalous {
				return 1
			}
			return 0
		default:
			return float64(row.RequestCount)
		}
	}

	return func(a, b EndpointRow) bool {
		if sortKey == "" {
			if a.Anomalous != b.Anomalous {
				return a.Anomalous
			}
			if a.RequestCount != b.RequestCount {
				return a.RequestCount > b.RequestCount
			}
			if a.Service != b.Service {
				return a.Service < b.Service
			}
			if a.Endpoint != b.Endpoint {
				return a.Endpoint < b.Endpoint
			}
			return a.Method < b.Method
		}
		if sortKey == "service" || sortKey == "endpoint" || sortKey == "method" {
			var as, bs string
			switch sortKey {
			case "service":
				as, bs = a.Service, b.Service
			case "endpoint":
				as, bs = a.Endpoint, b.Endpoint
			default:
				as, bs = a.Method, b.Method
			}
			if as == bs {
				return a.RequestCount > b.RequestCount
			}
			if desc {
				return as > bs
			}
			return as < bs
		}
		av, bv := val(a), val(b)
		if av == bv {
			return a.RequestCount > b.RequestCount
		}
		if desc {
			return av > bv
		}
		return av < bv
	}
}
