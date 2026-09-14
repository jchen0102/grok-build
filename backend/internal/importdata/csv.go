package importdata

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jchen0102/grok-build/internal/store"
)

const sampleCap = 5

var requiredHeaders = []string{
	"request_id",
	"timestamp",
	"service",
	"endpoint",
	"method",
	"status_code",
	"latency_ms",
}

type FailureClass string

const (
	MissingColumn   FailureClass = "missing_column"
	InvalidTime     FailureClass = "invalid_time"
	InvalidStatus   FailureClass = "invalid_status"
	InvalidLatency  FailureClass = "invalid_latency"
	DuplicateInFile FailureClass = "duplicate_in_file"
	DuplicateInDB   FailureClass = "duplicate_in_db"
)

type Sample struct {
	Line      int    `json:"line"`
	RequestID string `json:"request_id,omitempty"`
	Detail    string `json:"detail"`
}

type FailureBucket struct {
	Count   int      `json:"count"`
	Samples []Sample `json:"samples"`
}

type Summary struct {
	Success   int `json:"success"`
	Duplicate int `json:"duplicate"`
	Failed    int `json:"failed"`
}

type Result struct {
	Summary  Summary                         `json:"summary"`
	Failures map[FailureClass]*FailureBucket `json:"failures"`
}

type HeaderError struct {
	Message string
}

func (e *HeaderError) Error() string { return e.Message }

type ParsedRow struct {
	store.Request
	Line int
}

func newResult() *Result {
	return &Result{
		Failures: map[FailureClass]*FailureBucket{
			MissingColumn:   {Samples: []Sample{}},
			InvalidTime:     {Samples: []Sample{}},
			InvalidStatus:   {Samples: []Sample{}},
			InvalidLatency:  {Samples: []Sample{}},
			DuplicateInFile: {Samples: []Sample{}},
			DuplicateInDB:   {Samples: []Sample{}},
		},
	}
}

func (r *Result) add(class FailureClass, line int, requestID, detail string) {
	bucket := r.Failures[class]
	bucket.Count++
	if len(bucket.Samples) < sampleCap {
		bucket.Samples = append(bucket.Samples, Sample{Line: line, RequestID: requestID, Detail: detail})
	}
	switch class {
	case DuplicateInFile, DuplicateInDB:
		r.Summary.Duplicate++
	default:
		r.Summary.Failed++
	}
}

func Parse(r io.Reader) ([]ParsedRow, *Result, error) {
	reader := csv.NewReader(r)
	reader.ReuseRecord = false
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err == io.EOF {
		return nil, nil, &HeaderError{Message: "CSV 缺少表头"}
	}
	if err != nil {
		return nil, nil, &HeaderError{Message: "无法读取 CSV 表头"}
	}

	index, err := headerIndex(header)
	if err != nil {
		return nil, nil, err
	}

	result := newResult()
	seen := make(map[string]int)
	var rows []ParsedRow
	line := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			result.add(MissingColumn, line, "", "CSV 行无法解析")
			continue
		}

		get := func(name string) string {
			i := index[name]
			if i >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[i])
		}

		requestID := get("request_id")
		tsRaw := get("timestamp")
		service := get("service")
		endpoint := get("endpoint")
		method := get("method")
		statusRaw := get("status_code")
		latencyRaw := get("latency_ms")

		if requestID == "" {
			result.add(MissingColumn, line, "", "request_id 为空")
			continue
		}
		if service == "" {
			result.add(MissingColumn, line, requestID, "service 为空")
			continue
		}
		if endpoint == "" {
			result.add(MissingColumn, line, requestID, "endpoint 为空")
			continue
		}
		if method == "" {
			result.add(MissingColumn, line, requestID, "method 为空")
			continue
		}
		if tsRaw == "" {
			result.add(MissingColumn, line, requestID, "timestamp 为空")
			continue
		}
		if statusRaw == "" {
			result.add(MissingColumn, line, requestID, "status_code 为空")
			continue
		}
		if latencyRaw == "" {
			result.add(MissingColumn, line, requestID, "latency_ms 为空")
			continue
		}

		ts, err := parseUTC(tsRaw)
		if err != nil {
			result.add(InvalidTime, line, requestID, err.Error())
			continue
		}

		status, err := strconv.Atoi(statusRaw)
		if err != nil || status < 100 || status > 599 {
			result.add(InvalidStatus, line, requestID, "状态码必须是 100–599 的整数")
			continue
		}

		latency, err := strconv.Atoi(latencyRaw)
		if err != nil || latency < 0 {
			result.add(InvalidLatency, line, requestID, "延迟必须是 >= 0 的整数毫秒")
			continue
		}

		if firstLine, ok := seen[requestID]; ok {
			result.add(DuplicateInFile, line, requestID, fmt.Sprintf("与第 %d 行重复，先到先得", firstLine))
			continue
		}
		seen[requestID] = line

		rows = append(rows, ParsedRow{
			Request: store.Request{
				RequestID:  requestID,
				TS:         ts,
				Service:    service,
				Endpoint:   endpoint,
				Method:     method,
				StatusCode: status,
				LatencyMS:  latency,
			},
			Line: line,
		})
	}

	return rows, result, nil
}

func ApplyDBDuplicates(result *Result, rows []ParsedRow, dupIDs []string) {
	lineByID := make(map[string]int, len(rows))
	for _, row := range rows {
		lineByID[row.RequestID] = row.Line
	}
	for _, id := range dupIDs {
		result.add(DuplicateInDB, lineByID[id], id, "库内已存在相同 request_id")
	}
}

func headerIndex(header []string) (map[string]int, error) {
	index := make(map[string]int, len(header))
	for i, col := range header {
		name := strings.ToLower(strings.TrimSpace(col))
		index[name] = i
	}
	var missing []string
	for _, name := range requiredHeaders {
		if _, ok := index[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, &HeaderError{Message: "CSV 缺少表头列: " + strings.Join(missing, ", ")}
	}
	return index, nil
}

func parseUTC(raw string) (time.Time, error) {
	layouts := []string{time.RFC3339Nano, time.RFC3339}
	var ts time.Time
	var err error
	for _, layout := range layouts {
		ts, err = time.Parse(layout, raw)
		if err == nil {
			if ts.Location() == time.Local {
				return time.Time{}, fmt.Errorf("时间必须带时区（UTC）")
			}
			return ts.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("时间非法，需要 UTC RFC3339")
}
