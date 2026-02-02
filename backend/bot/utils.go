package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// parseDateRangeFromParams parses day=, start=, end= from query params. If only day is set, returns that day. If start/end are set, returns the range. If none, returns error.
func parseDateRangeFromParams(r *http.Request) (time.Time, time.Time, error) {
	day := r.URL.Query().Get("day")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	layout := "2006-01-02"
	if day != "" {
		t, err := time.Parse(layout, day)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		return t, t, nil
	}
	if startStr != "" && endStr != "" {
		start, err1 := time.Parse(layout, startStr)
		end, err2 := time.Parse(layout, endStr)
		if err1 != nil || err2 != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start or end date")
		}
		if start.After(end) {
			return time.Time{}, time.Time{}, fmt.Errorf("start date must not be after end date")
		}
		return start, end, nil
	}
	return time.Time{}, time.Time{}, fmt.Errorf("must provide either day=YYYY-MM-DD or start=YYYY-MM-DD&end=YYYY-MM-DD")
}

// parseSlackTimestamp parses Slack timestamps of the form "1234567890.123456"
// and returns a time.Time preserving fractional seconds.
func parseSlackTimestamp(ts string) (time.Time, error) {
	// Slack ts format: seconds[.fraction]
	var secPart string
	var fracPart string
	if idx := strings.IndexByte(ts, '.'); idx >= 0 {
		secPart = ts[:idx]
		fracPart = ts[idx+1:]
	} else {
		secPart = ts
	}
	secs, err := strconv.ParseInt(secPart, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	nsec := int64(0)
	if fracPart != "" {
		// pad or trim to nanoseconds
		if len(fracPart) > 9 {
			fracPart = fracPart[:9]
		} else {
			for len(fracPart) < 9 {
				fracPart += "0"
			}
		}
		if nf, err := strconv.ParseInt(fracPart, 10, 64); err == nil {
			nsec = nf
		}
	}
	return time.Unix(secs, nsec), nil
}
