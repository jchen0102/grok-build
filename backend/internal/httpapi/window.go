package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jchen0102/grok-build/internal/store"
)

func parseWindow(r *http.Request, st *store.Store) (from, to time.Time, emptyDB bool, err error) {
	fromRaw := r.URL.Query().Get("from")
	toRaw := r.URL.Query().Get("to")
	if (fromRaw == "") != (toRaw == "") {
		return time.Time{}, time.Time{}, false, fmt.Errorf("from 和 to 必须成对提供")
	}

	if fromRaw == "" {
		minTS, maxTS, boundErr := st.TimeBounds(r.Context())
		if boundErr != nil {
			return time.Time{}, time.Time{}, false, boundErr
		}
		if minTS == nil || maxTS == nil {
			return time.Time{}, time.Time{}, true, nil
		}
		return minTS.UTC(), maxTS.UTC(), false, nil
	}

	from, err = parseUTC(fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, false, err
	}
	to, err = parseUTC(toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, false, err
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, false, fmt.Errorf("from 不能晚于 to")
	}
	return from, to, false, nil
}

func parseUTC(raw string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("时间必须是 UTC RFC3339")
}
