package domain

import (
	"context"
	"errors"
	"testing"
)

type fakeResolver struct {
	detail string
	err    error
}

func (f fakeResolver) ResolveToDetailURL(ctx context.Context, raw string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.detail, nil
}

type fakeAuth struct {
	token string
	err   error
}

func (f fakeAuth) RetrieveToken(ctx context.Context, areaID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.token, nil
}

type fakeProgram struct {
	meta ProgramMeta
	err  error
}

func (f fakeProgram) ResolveProgramMeta(ctx context.Context, stationID, ft string) (ProgramMeta, error) {
	if f.err != nil {
		return ProgramMeta{}, f.err
	}
	return f.meta, nil
}

type fakePlaylist struct {
	urls []string
	err  error
}

func (f fakePlaylist) BuildSegmentURLs(ctx context.Context, in SegmentInput) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.urls, nil
}

type fakeAudio struct {
	out string
	err error
}

func (f fakeAudio) DownloadAndMergeAacSegments(ctx context.Context, urls []string, outputDir, fileName string, onProgress func(done, total int)) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if onProgress != nil {
		onProgress(len(urls), len(urls))
	}
	return f.out, nil
}

func TestDownloaderResolvePassThrough(t *testing.T) {
	d := &Downloader{resolver: fakeResolver{detail: "x"}}
	got, err := d.ResolveToDetailURL(context.Background(), "in")
	if err != nil || got != "x" {
		t.Fatalf("unexpected result: %q %v", got, err)
	}
}

func TestDownloaderDownloadFromDetailURLSuccessWithGivenArea(t *testing.T) {
	d := &Downloader{
		resolver: fakeResolver{detail: "https://radiko.jp/#!/ts/AAA/20260101000000"},
		resolveAreaID: func(ctx context.Context, stationID string) (string, error) {
			return "JP99", errors.New("should not be called")
		},
		auth:     fakeAuth{token: "tok"},
		program:  fakeProgram{meta: ProgramMeta{FT: "20260101000000", TO: "20260101050000", Title: "T"}},
		playlist: fakePlaylist{urls: []string{"u1", "u2"}},
		audio:    fakeAudio{out: "out.aac"},
	}
	got, err := d.DownloadFromDetailURL(context.Background(), "https://radiko.jp/#!/ts/AAA/20260101000000", DownloadOptions{AreaID: "JP1", OutputDir: t.TempDir()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "out.aac" {
		t.Fatalf("want out.aac, got %s", got)
	}
}

func TestDownloaderDownloadFromDetailURLNoSegments(t *testing.T) {
	d := &Downloader{
		resolveAreaID: func(ctx context.Context, stationID string) (string, error) { return "JP1", nil },
		auth:          fakeAuth{token: "tok"},
		program:       fakeProgram{meta: ProgramMeta{FT: "20260101000000", TO: "20260101050000", Title: "T"}},
		playlist:      fakePlaylist{urls: nil},
		audio:         fakeAudio{out: "out.aac"},
	}
	_, err := d.DownloadFromDetailURL(context.Background(), "https://radiko.jp/#!/ts/AAA/20260101000000", DownloadOptions{OutputDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected no segments error")
	}
}

func TestDownloaderDownloadFromDetailURLPropagatesErrors(t *testing.T) {
	tests := []struct {
		name string
		d    *Downloader
	}{
		{name: "area", d: &Downloader{resolveAreaID: func(ctx context.Context, stationID string) (string, error) { return "", errors.New("area") }}},
		{name: "auth", d: &Downloader{resolveAreaID: func(ctx context.Context, stationID string) (string, error) { return "JP1", nil }, auth: fakeAuth{err: errors.New("auth")}}},
		{name: "program", d: &Downloader{resolveAreaID: func(ctx context.Context, stationID string) (string, error) { return "JP1", nil }, auth: fakeAuth{token: "tok"}, program: fakeProgram{err: errors.New("program")}}},
		{name: "playlist", d: &Downloader{resolveAreaID: func(ctx context.Context, stationID string) (string, error) { return "JP1", nil }, auth: fakeAuth{token: "tok"}, program: fakeProgram{meta: ProgramMeta{FT: "20260101000000", TO: "20260101050000", Title: "T"}}, playlist: fakePlaylist{err: errors.New("playlist")}}},
		{name: "audio", d: &Downloader{resolveAreaID: func(ctx context.Context, stationID string) (string, error) { return "JP1", nil }, auth: fakeAuth{token: "tok"}, program: fakeProgram{meta: ProgramMeta{FT: "20260101000000", TO: "20260101050000", Title: "T"}}, playlist: fakePlaylist{urls: []string{"u1"}}, audio: fakeAudio{err: errors.New("audio")}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.d.DownloadFromDetailURL(context.Background(), "https://radiko.jp/#!/ts/AAA/20260101000000", DownloadOptions{OutputDir: t.TempDir()})
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
