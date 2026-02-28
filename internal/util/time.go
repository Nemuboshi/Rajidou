package util

import (
	"fmt"
	"time"
)

// Source map in this file:
// - StepTimestamp behavior is adapted from rajiko/modules/timeshift.js seek stepping.
// - Parse/Format helpers are CLI utilities.
const tsLayout = "20060102150405"

// ParseTimestamp parses a local-time timestamp in "YYYYMMDDHHMMSS" format.
//
// Input must be exactly 14 digits and is interpreted in time.Local.
func ParseTimestamp(ts string) (time.Time, error) {
	if len(ts) != 14 {
		return time.Time{}, fmt.Errorf("invalid timestamp: %s", ts)
	}
	t, err := time.ParseInLocation(tsLayout, ts, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp: %s", ts)
	}
	return t, nil
}

// FormatTimestamp formats t as "YYYYMMDDHHMMSS".
func FormatTimestamp(t time.Time) string {
	return t.Format(tsLayout)
}

// StepTimestamp shifts a "YYYYMMDDHHMMSS" timestamp by seconds and returns the
// same layout.
func StepTimestamp(ts string, seconds int) (string, error) {
	t, err := ParseTimestamp(ts)
	if err != nil {
		return "", err
	}
	return FormatTimestamp(t.Add(time.Duration(seconds) * time.Second)), nil
}
