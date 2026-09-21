package metrics

import (
	"fmt"
	"testing"
)

func ptrInt(v int) *int { return &v }

func TestP95(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   []int
		want *int
	}{
		{name: "n=0 is null", in: nil, want: nil},
		{name: "n=1 is the only value", in: []int{42}, want: ptrInt(42)},
		{name: "n=20 nearest-rank ceil is 19th", in: seq(1, 20), want: ptrInt(19)},
		{name: "n=20 unsorted still 19th after sort", in: []int{20, 1, 19, 2, 18, 3, 17, 4, 16, 5, 15, 6, 14, 7, 13, 8, 12, 9, 11, 10}, want: ptrInt(19)},
		{name: "n=19 nearest-rank clamps to last", in: seq(1, 19), want: ptrInt(19)},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := P95(tc.in)
			if tc.want == nil {
				if got != nil {
					t.Fatalf("P95(%v) = %d, want nil", tc.in, *got)
				}
				return
			}
			if got == nil || *got != *tc.want {
				t.Fatalf("P95(%v) = %v, want %d", tc.in, fmtPtr(got), *tc.want)
			}
		})
	}
}

func TestAnomaly(t *testing.T) {
	t.Parallel()
	rate := func(v float64) *float64 { return &v }
	cases := []struct {
		name    string
		n       int
		rate    *float64
		p95     *int
		want    bool
		wantWhy []string
	}{
		{name: "n=19 never anomalous even if both trip", n: 19, rate: rate(1), p95: ptrInt(5000), want: false, wantWhy: []string{}},
		{name: "n=20 high 5xx only", n: 20, rate: rate(0.10), p95: ptrInt(200), want: true, wantWhy: []string{ReasonHigh5xx}},
		{name: "n=20 high p95 only", n: 20, rate: rate(0.09), p95: ptrInt(1000), want: true, wantWhy: []string{ReasonHighP95}},
		{name: "n=20 dual reasons", n: 20, rate: rate(0.10), p95: ptrInt(1000), want: true, wantWhy: []string{ReasonHigh5xx, ReasonHighP95}},
		{name: "n=20 neither threshold", n: 20, rate: rate(0.099), p95: ptrInt(999), want: false, wantWhy: []string{}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyAnomaly(tc.n, tc.rate, tc.p95)
			if got.Anomalous != tc.want {
				t.Fatalf("anomalous=%v, want %v", got.Anomalous, tc.want)
			}
			if len(got.Reasons) != len(tc.wantWhy) {
				t.Fatalf("reasons=%v, want %v", got.Reasons, tc.wantWhy)
			}
			for i := range tc.wantWhy {
				if got.Reasons[i] != tc.wantWhy[i] {
					t.Fatalf("reasons=%v, want %v", got.Reasons, tc.wantWhy)
				}
			}
		})
	}
}

func TestSummarizeCountEqualsRowsAnd4xxNotError(t *testing.T) {
	t.Parallel()
	statuses := make([]int, 20)
	latencies := make([]int, 20)
	for i := 0; i < 20; i++ {
		latencies[i] = i + 1
		statuses[i] = 200
	}
	statuses[0] = 404
	statuses[1] = 500
	stat := SummarizeEndpoint(EndpointKey{Service: "s", Endpoint: "/x", Method: "GET"}, statuses, latencies)
	if stat.RequestCount != 20 {
		t.Fatalf("request count=%d, want 20 (must equal row count)", stat.RequestCount)
	}
	if stat.FiveXXCount != 1 {
		t.Fatalf("5xx count=%d, want 1 (4xx is not an error)", stat.FiveXXCount)
	}
	if stat.FiveXXRate == nil || *stat.FiveXXRate != 0.05 {
		t.Fatalf("5xx rate=%v, want 0.05", fmtFloat(stat.FiveXXRate))
	}
	if stat.Anomalous {
		t.Fatal("4xx must not create an anomaly")
	}
}

func seq(from, to int) []int {
	out := make([]int, 0, to-from+1)
	for i := from; i <= to; i++ {
		out = append(out, i)
	}
	return out
}

func fmtPtr(v *int) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *v)
}

func fmtFloat(v *float64) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("%v", *v)
}
