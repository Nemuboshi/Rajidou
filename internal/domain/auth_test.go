package domain

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"rajidou/internal/netx"
)

func TestAuthCacheSaveAndLoad(t *testing.T) {
	net := netx.NewClient(time.Second, netx.RetryOptions{Retries: 1, BaseDelay: time.Millisecond})
	a := NewAuthClient(net)
	a.cachePath = filepath.Join(t.TempDir(), "auth.json")
	a.tokenCache["JP1"] = TokenCacheItem{Token: "tok", RequestTime: time.Now().UnixMilli()}
	if err := a.saveCache(); err != nil {
		t.Fatalf("save cache: %v", err)
	}

	b := NewAuthClient(net)
	b.cachePath = a.cachePath
	b.ensureCacheLoaded()
	if got := b.tokenCache["JP1"].Token; got != "tok" {
		t.Fatalf("want tok, got %s", got)
	}
}

func TestAuthCacheLoadBrokenFile(t *testing.T) {
	net := netx.NewClient(time.Second, netx.RetryOptions{Retries: 1, BaseDelay: time.Millisecond})
	a := NewAuthClient(net)
	a.cachePath = filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(a.cachePath, []byte("{"), 0o644); err != nil {
		t.Fatalf("write broken: %v", err)
	}
	a.ensureCacheLoaded()
	if len(a.tokenCache) != 0 {
		t.Fatalf("expected empty cache, got %+v", a.tokenCache)
	}
}

func TestResolveStationAreaIDNotFoundWithCanceledContext(t *testing.T) {
	net := netx.NewClient(time.Second, netx.RetryOptions{Retries: 0, BaseDelay: time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ResolveStationAreaID(ctx, net, "NOPE")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRetrieveTokenUsesCache(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("network should not be called on fresh cache")
	})
	defer closeFn()

	a := NewAuthClient(net)
	a.tokenCache["JP1"] = TokenCacheItem{Token: "cached", RequestTime: time.Now().UnixMilli()}
	token, err := a.RetrieveToken(context.Background(), "JP1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "cached" {
		t.Fatalf("want cached token, got %s", token)
	}
}

func TestRetrieveTokenSuccess(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/api/auth1":
			w.Header().Set("x-radiko-authtoken", "tok")
			w.Header().Set("x-radiko-keyoffset", "0")
			w.Header().Set("x-radiko-keylength", "8")
			w.WriteHeader(http.StatusOK)
		case "/v2/api/auth2":
			if r.Header.Get("X-Radiko-Partialkey") == "" {
				t.Fatal("missing partial key")
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})
	defer closeFn()

	a := NewAuthClient(net)
	a.cachePath = filepath.Join(t.TempDir(), "auth.json")
	token, err := a.RetrieveToken(context.Background(), "JP1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "tok" {
		t.Fatalf("want tok, got %s", token)
	}
}

func TestRetrieveTokenAuth1Errors(t *testing.T) {
	tests := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{
			name: "non-2xx",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
			},
		},
		{
			name: "missing-headers",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("x-radiko-authtoken", "tok")
				w.WriteHeader(http.StatusOK)
			},
		},
		{
			name: "invalid-key-range",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("x-radiko-authtoken", "tok")
				w.Header().Set("x-radiko-keyoffset", "999999")
				w.Header().Set("x-radiko-keylength", "10")
				w.WriteHeader(http.StatusOK)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v2/api/auth1" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				tt.handler(w, r)
			})
			defer closeFn()

			a := NewAuthClient(net)
			a.cachePath = filepath.Join(t.TempDir(), "auth.json")
			if _, err := a.RetrieveToken(context.Background(), "JP1"); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestRetrieveTokenAuth2ErrorAndBadArea(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/api/auth1":
			w.Header().Set("x-radiko-authtoken", "tok")
			w.Header().Set("x-radiko-keyoffset", "0")
			w.Header().Set("x-radiko-keylength", "8")
			w.WriteHeader(http.StatusOK)
		case "/v2/api/auth2":
			w.WriteHeader(http.StatusUnauthorized)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})
	defer closeFn()

	a := NewAuthClient(net)
	a.cachePath = filepath.Join(t.TempDir(), "auth.json")
	if _, err := a.RetrieveToken(context.Background(), "JP1"); err == nil {
		t.Fatal("expected auth2 error")
	}
	if _, err := a.RetrieveToken(context.Background(), "BAD"); err == nil {
		t.Fatal("expected area id error")
	}
}

func TestRetrieveTokenAuth2NetError(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/api/auth1" {
			w.Header().Set("x-radiko-authtoken", "tok")
			w.Header().Set("x-radiko-keyoffset", "0")
			w.Header().Set("x-radiko-keylength", "8")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, fmt.Sprintf("bad path: %s", r.URL.Path), http.StatusInternalServerError)
	})
	defer closeFn()

	a := NewAuthClient(net)
	a.cachePath = filepath.Join(t.TempDir(), "auth.json")
	if _, err := a.RetrieveToken(context.Background(), "JP1"); err == nil {
		t.Fatal("expected error")
	}
}
