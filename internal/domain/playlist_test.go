package domain

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func TestFirstDataLineNoData(t *testing.T) {
	_, err := firstDataLine("#EXTM3U\n#EXTINF:1\n")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAllDataLines(t *testing.T) {
	got := allDataLines("#A\nline1\n\n#B\n line2 \n")
	if len(got) != 2 || got[0] != "line1" || got[1] != "line2" {
		t.Fatalf("unexpected lines: %#v", got)
	}
}

func TestBuildSegmentURLsSuccess(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/station/stream/pc_html5/AAA.xml":
			_, _ = fmt.Fprint(w, `<root><playlist_create_url>https://radiko.jp/tf/playlist.m3u8</playlist_create_url></root>`)
		case "/tf/playlist.m3u8":
			_, _ = fmt.Fprint(w, "#EXTM3U\nhttps://radiko.jp/chunk.m3u8\n")
		case "/chunk.m3u8":
			_, _ = fmt.Fprint(w, "#EXTM3U\nhttps://radiko.jp/a.aac\nhttps://radiko.jp/b.aac\n")
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})
	defer closeFn()

	p := NewPlaylistBuilder(net)
	urls, err := p.BuildSegmentURLs(context.Background(), SegmentInput{
		StationID: "AAA",
		FT:        "20260219000000",
		TO:        "20260219000500",
		Token:     "tok",
		AreaID:    "JP1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 2 {
		t.Fatalf("want 2 urls, got %d (%v)", len(urls), urls)
	}
}

func TestBuildSegmentURLsExpired(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/station/stream/pc_html5/AAA.xml":
			_, _ = fmt.Fprint(w, `<root><playlist_create_url>https://radiko.jp/tf/playlist.m3u8</playlist_create_url></root>`)
		case "/tf/playlist.m3u8":
			w.WriteHeader(http.StatusForbidden)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})
	defer closeFn()

	p := NewPlaylistBuilder(net)
	_, err := p.BuildSegmentURLs(context.Background(), SegmentInput{
		StationID: "AAA",
		FT:        "20260219000000",
		TO:        "20260219000500",
		Token:     "tok",
		AreaID:    "JP1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildSegmentURLsInvalidTO(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v3/station/stream/pc_html5/AAA.xml" {
			_, _ = fmt.Fprint(w, `<root></root>`)
			return
		}
		t.Fatalf("unexpected path: %s", r.URL.Path)
	})
	defer closeFn()

	p := NewPlaylistBuilder(net)
	_, err := p.BuildSegmentURLs(context.Background(), SegmentInput{
		StationID: "AAA",
		FT:        "20260219000000",
		TO:        "bad",
		Token:     "tok",
		AreaID:    "JP1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
