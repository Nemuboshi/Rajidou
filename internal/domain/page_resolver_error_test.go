package domain

import (
	"context"
	"net/http"
	"testing"
)

func TestBuildDetailURLsFromSearchAPIDataInvalidJSON(t *testing.T) {
	_, err := BuildDetailURLsFromSearchAPIData([]byte(`{`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestToFTTimestampInvalid(t *testing.T) {
	if got := toFTTimestamp("bad"); got != "" {
		t.Fatalf("want empty, got %s", got)
	}
}

func TestExtractSearchKeyFromURLInvalid(t *testing.T) {
	if got := extractSearchKeyFromURL("://bad"); got != "" {
		t.Fatalf("want empty, got %s", got)
	}
}

func TestResolveToDetailURLUnsupported(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {})
	defer closeFn()

	r := NewPageResolver(net)
	if _, err := r.ResolveToDetailURL(context.Background(), "https://example.com/abc"); err == nil {
		t.Fatal("expected unsupported link error")
	}
}

func TestResolveToDetailURLSearchNoResult(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	defer closeFn()

	r := NewPageResolver(net)
	if _, err := r.ResolveToDetailURL(context.Background(), "https://radiko.jp/#!/search/timeshift?key=x"); err == nil {
		t.Fatal("expected error")
	}
}
