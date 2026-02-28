package domain

import (
	"testing"
	"time"
)

func TestClassifyRadikoLink(t *testing.T) {
	search := "https://radiko.jp/#!/search/timeshift?key=sora%20to%20hoshi"
	if got := ClassifyRadikoLink(search); got != LinkKindSearch {
		t.Fatalf("expected search, got %q", got)
	}

	detail := "https://radiko.jp/#!/ts/ALPHA-STATION/20260219000000"
	if got := ClassifyRadikoLink(detail); got != LinkKindDetail {
		t.Fatalf("expected detail, got %q", got)
	}
}

func TestExtractDetailFromDetailURL(t *testing.T) {
	url := "https://radiko.jp/#!/ts/ALPHA-STATION/20260219000000"
	d, err := ExtractDetailFromDetailURL(url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.StationID != "ALPHA-STATION" || d.FT != "20260219000000" {
		t.Fatalf("unexpected detail: %+v", d)
	}
}

func TestPickLatestDetailURL(t *testing.T) {
	now := time.Date(2026, 2, 20, 0, 0, 0, 0, time.Local)
	links := []string{
		"https://radiko.jp/#!/ts/ALPHA-STATION/20260217000000",
		"https://radiko.jp/#!/ts/ALPHA-STATION/20260218000000",
		"https://radiko.jp/#!/ts/ALPHA-STATION/20260219000000",
	}
	got, err := PickLatestDetailURL(links, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://radiko.jp/#!/ts/ALPHA-STATION/20260219000000"
	if got != want {
		t.Fatalf("want %s, got %s", want, got)
	}
}
