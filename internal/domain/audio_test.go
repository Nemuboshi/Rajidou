package domain

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"rajidou/internal/netx"
)

func TestParseAACPackedHeaderSize(t *testing.T) {
	if got := ParseAACPackedHeaderSize([]byte{0x00}); got != 0 {
		t.Fatalf("want 0, got %d", got)
	}
	id3 := []byte{'I', 'D', '3', 0, 0, 0, 0, 0, 0, 3, 1, 2, 3, 9, 9}
	if got := ParseAACPackedHeaderSize(id3); got != 13 {
		t.Fatalf("want 13, got %d", got)
	}
}

func TestDownloadAndMergeAacSegments(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a":
			_, _ = w.Write([]byte{'I', 'D', '3', 0, 0, 0, 0, 0, 0, 1, 0xAA, 0x01, 0x02})
		case "/b":
			_, _ = w.Write([]byte{0x03, 0x04})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer s.Close()

	net := netx.NewClient(2*time.Second, netx.RetryOptions{Retries: 1, BaseDelay: time.Millisecond})
	d := NewAudioDownloader(net, 2)
	tmp := t.TempDir()
	var progress int32
	out, err := d.DownloadAndMergeAacSegments(context.Background(), []string{s.URL + "/a", s.URL + "/b"}, tmp, "x.aac", func(done, total int) {
		if done == total {
			atomic.StoreInt32(&progress, 1)
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&progress) != 1 {
		t.Fatal("expected progress callback")
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	want := []byte{0x01, 0x02, 0x03, 0x04}
	if !bytes.Equal(b, want) {
		t.Fatalf("want %v, got %v", want, b)
	}
	if filepath.Ext(out) != ".aac" {
		t.Fatalf("unexpected output file: %s", out)
	}
}

func TestDownloadAndMergeAacSegmentsError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer s.Close()

	net := netx.NewClient(2*time.Second, netx.RetryOptions{Retries: 0, BaseDelay: time.Millisecond})
	d := NewAudioDownloader(net, 1)
	_, err := d.DownloadAndMergeAacSegments(context.Background(), []string{s.URL + "/bad"}, t.TempDir(), "x.aac", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
