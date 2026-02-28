package domain

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"rajidou/internal/netx"
)

func TestBuildDetailURLsFromSearchAPIData(t *testing.T) {
	payload := []byte(`{"data":[{"station_id":"ALPHA-STATION","start_time":"2026-02-19 00:00:00"}]}`)
	urls, err := BuildDetailURLsFromSearchAPIData(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 1 || urls[0] != "https://radiko.jp/#!/ts/ALPHA-STATION/20260219000000" {
		t.Fatalf("unexpected urls: %#v", urls)
	}
}

func TestResolveToDetailURLKeepsDetailURL(t *testing.T) {
	net := netx.NewClient(2*time.Second, netx.RetryOptions{Retries: 1, BaseDelay: time.Millisecond})
	r := NewPageResolver(net)
	input := "https://radiko.jp/#!/ts/ALPHA-STATION/20260219000000"
	got, err := r.ResolveToDetailURL(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Fatalf("want %s, got %s", input, got)
	}
}

func TestExtractSearchKeyFromURL(t *testing.T) {
	key := extractSearchKeyFromURL("https://radiko.jp/#!/search/timeshift?key=%E3%83%93%E3%82%BF%E3%83%9F%E3%83%B3M")
	if key == "" {
		t.Fatal("expected non-empty key")
	}
}

func TestResolveToDetailURLFromSearch(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/programs/legacy/perl/program/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"data":[{"station_id":"AAA","start_time":"2000-01-01 00:00:00"},{"station_id":"AAA","start_time":"2099-01-01 00:00:00"}]}`)
	})
	defer closeFn()

	r := NewPageResolver(net)
	got, err := r.ResolveToDetailURL(context.Background(), "https://radiko.jp/#!/search/timeshift?key=%E3%83%86%E3%82%B9%E3%83%88")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://radiko.jp/#!/ts/AAA/20000101000000" {
		t.Fatalf("unexpected detail url: %s", got)
	}
}

func TestFetchDetailLinksFromSearchAPIEmptyKey(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("request should not be sent for empty key")
	})
	defer closeFn()

	r := NewPageResolver(net)
	got, err := r.fetchDetailLinksFromSearchAPI(context.Background(), "https://radiko.jp/#!/search/timeshift")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("want nil result, got %v", got)
	}
}

func TestNetClientGetText(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	defer s.Close()

	net := netx.NewClient(2*time.Second, netx.RetryOptions{Retries: 1, BaseDelay: time.Millisecond})
	status, body, err := net.GetText(context.Background(), s.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 || body != "ok" {
		t.Fatalf("unexpected response: %d %q", status, body)
	}
}
