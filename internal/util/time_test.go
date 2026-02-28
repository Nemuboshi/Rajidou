package util

import (
	"testing"
	"time"
)

func TestStepTimestamp(t *testing.T) {
	got, err := StepTimestamp("20260219000000", 300)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "20260219000500" {
		t.Fatalf("want 20260219000500, got %s", got)
	}
}

func TestParseTimestampErrors(t *testing.T) {
	if _, err := ParseTimestamp("short"); err == nil {
		t.Fatal("expected error for invalid length")
	}
	if _, err := ParseTimestamp("20261301120000"); err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestFormatTimestamp(t *testing.T) {
	got := FormatTimestamp(time.Date(2026, 2, 19, 12, 34, 56, 0, time.Local))
	if got != "20260219123456" {
		t.Fatalf("want 20260219123456, got %s", got)
	}
}
