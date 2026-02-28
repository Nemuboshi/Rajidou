package domain

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"rajidou/internal/netx"
)

type rewriteTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	u := *clone.URL
	u.Scheme = t.target.Scheme
	u.Host = t.target.Host
	clone.URL = &u
	clone.Host = t.target.Host
	return t.base.RoundTrip(clone)
}

func newMockNetClient(t *testing.T, handler http.HandlerFunc) (*netx.Client, func()) {
	t.Helper()
	s := httptest.NewTLSServer(handler)
	target, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	httpClient := s.Client()
	httpClient.Timeout = 2 * time.Second
	httpClient.Transport = &rewriteTransport{base: httpClient.Transport, target: target}
	return netx.NewClientWithHTTPClient(httpClient, netx.RetryOptions{Retries: 0, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}), s.Close
}

func TestLoadRajikoAppKey(t *testing.T) {
	k := LoadRajikoAppKey()
	if k.AppVersion != RadikoAppVersion || k.AppID != RadikoAppID || len(k.AppKeyBase64) <= 1000 {
		t.Fatalf("unexpected key material: %+v", k)
	}
}

func TestRadikoAppKeyBase64IsDecodable(t *testing.T) {
	k := LoadRajikoAppKey()
	if _, err := base64.StdEncoding.DecodeString(k.AppKeyBase64); err != nil {
		t.Fatalf("app key must be valid base64: %v", err)
	}
}

func TestExtractStationIDsFromAreaXML(t *testing.T) {
	xml := `<stations><station><id>AAA</id></station><station><id> BBB </id></station></stations>`
	got := extractStationIDsFromAreaXML(xml)
	want := []string{"AAA", "BBB"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestGenGPSAndDeviceInfo(t *testing.T) {
	gps, err := GenGPS("JP1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(gps, ",gps") {
		t.Fatalf("unexpected gps: %s", gps)
	}
	if _, err := GenGPS("BAD"); err == nil {
		t.Fatal("expected error")
	}

	appVer, userID, userAgent, device := GenDeviceInfo("1.0.0")
	if appVer != "1.0.0" || len(userID) != 32 || userAgent == "" || device == "" {
		t.Fatalf("unexpected device info: %q %q %q %q", appVer, userID, userAgent, device)
	}
}

func TestSleepClampAndRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	sleepContext(ctx, time.Second)
	if time.Since(start) > 200*time.Millisecond {
		t.Fatalf("sleepContext should return quickly on canceled context")
	}

	if got := clamp(10, 0, 5); got != 5 {
		t.Fatalf("want 5, got %v", got)
	}
	if got := clamp(-1, 0, 5); got != 0 {
		t.Fatalf("want 0, got %v", got)
	}

	req, err := newRequest(context.Background(), http.MethodGet, "https://example.com", map[string]string{"X-A": "B"})
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	if req.Header.Get("X-A") != "B" {
		t.Fatalf("header not set")
	}
}

func TestResolveStationAreaIDSuccessAndWarmAreaStations(t *testing.T) {
	stationAreaCache = sync.Map{}
	stationAreaWarm = sync.Once{}

	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v3/station/list/JP1.xml" {
			_, _ = fmt.Fprint(w, `<stations><station><id>ST001</id></station></stations>`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer closeFn()

	ok := warmAreaStations(context.Background(), net, "JP1")
	if !ok {
		t.Fatal("warmAreaStations should succeed")
	}
	area, err := ResolveStationAreaID(context.Background(), net, "ST001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if area != "JP1" {
		t.Fatalf("want JP1, got %s", area)
	}
}
