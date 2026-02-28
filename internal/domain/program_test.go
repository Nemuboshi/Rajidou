package domain

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func TestDecodeXML(t *testing.T) {
	got := decodeXML("A&amp;B &lt;ok&gt; &quot;x&quot; &#39;y&#39;")
	want := "A&B <ok> \"x\" 'y'"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestResolveProgramMetaSuccess(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/program/v3/weekly/AAA.xml" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `<radiko><prog ft="20260219000000" to="20260219003000"><title>A&amp;B</title></prog></radiko>`)
	})
	defer closeFn()

	r := NewProgramResolver(net)
	meta, err := r.ResolveProgramMeta(context.Background(), "AAA", "20260219000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.TO != "20260219003000" || meta.Title != "A&B" {
		t.Fatalf("unexpected meta: %+v", meta)
	}
}

func TestResolveProgramMetaStatusError(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	defer closeFn()

	r := NewProgramResolver(net)
	if _, err := r.ResolveProgramMeta(context.Background(), "AAA", "20260219000000"); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveProgramMetaMissingProgram(t *testing.T) {
	net, closeFn := newMockNetClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<radiko><prog ft="20260218000000" to="20260218003000"></prog></radiko>`)
	})
	defer closeFn()

	r := NewProgramResolver(net)
	if _, err := r.ResolveProgramMeta(context.Background(), "AAA", "20260219000000"); err == nil {
		t.Fatal("expected error")
	}
}
