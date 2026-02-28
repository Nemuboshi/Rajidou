package domain

import (
	"testing"
	"time"
)

func TestExtractDetailFromDetailURLInvalid(t *testing.T) {
	_, err := ExtractDetailFromDetailURL("https://radiko.jp/#!/ts/AAA/not-ts")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPickLatestDetailURLNoUsableEntry(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	_, err := PickLatestDetailURL([]string{"https://radiko.jp/#!/ts/AAA/20270201000000"}, now)
	if err == nil {
		t.Fatal("expected error")
	}
}
