package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"rajidou/internal/config"
	"rajidou/internal/domain"
	"rajidou/internal/netx"
)

func TestFormatError(t *testing.T) {
	if got := formatError(nil); got != "" {
		t.Fatalf("want empty string, got %q", got)
	}
	err := errors.New("boom")
	if got := formatError(err); got != "boom" {
		t.Fatalf("want boom, got %q", got)
	}
}

type fakeLogger struct{}

func (fakeLogger) Info(string)    {}
func (fakeLogger) Warn(string)    {}
func (fakeLogger) Error(string)   {}
func (fakeLogger) Success(string) {}
func (fakeLogger) Failure(string) {}

type fakeDownloader struct {
	resolveErr  error
	downloadErr error
}

func (f fakeDownloader) ResolveToDetailURL(ctx context.Context, raw string) (string, error) {
	if f.resolveErr != nil {
		return "", f.resolveErr
	}
	return "https://radiko.jp/#!/ts/AAA/20260101000000", nil
}

func (f fakeDownloader) DownloadFromDetailURL(ctx context.Context, detailURL string, opt domain.DownloadOptions) (string, error) {
	if f.downloadErr != nil {
		return "", f.downloadErr
	}
	return filepath.Join(opt.OutputDir, "x.aac"), nil
}

func TestExecuteConfigError(t *testing.T) {
	code := execute([]string{"--config", "x.yaml"}, fakeLogger{}, func(path string) (config.Config, error) {
		return config.Config{}, errors.New("bad config")
	}, fakeDownloader{})
	if code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestExecutePartialFailure(t *testing.T) {
	cfg := config.Config{
		Links:     []string{"a", "b"},
		OutputDir: t.TempDir(),
		Jobs:      1,
	}
	code := execute([]string{"--config", "x.yaml"}, fakeLogger{}, func(path string) (config.Config, error) {
		return cfg, nil
	}, fakeDownloader{
		downloadErr: errors.New("download fail"),
	})
	if code != 2 {
		t.Fatalf("want exit 2, got %d", code)
	}
}

func TestExecuteSuccess(t *testing.T) {
	cfg := config.Config{
		Links:     []string{"a"},
		OutputDir: t.TempDir(),
		Jobs:      1,
	}
	code := execute([]string{"--config", "x.yaml"}, fakeLogger{}, func(path string) (config.Config, error) {
		return cfg, nil
	}, fakeDownloader{})
	if code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
}

func TestExecuteResolveFailure(t *testing.T) {
	cfg := config.Config{
		Links:     []string{"a"},
		OutputDir: t.TempDir(),
		Jobs:      1,
	}
	code := execute([]string{"--config", "x.yaml"}, fakeLogger{}, func(path string) (config.Config, error) {
		return cfg, nil
	}, fakeDownloader{
		resolveErr: errors.New("resolve fail"),
	})
	if code != 2 {
		t.Fatalf("want exit 2, got %d", code)
	}
}

func TestExecuteInvalidConfigPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("filepath.Abs invalid-path behavior differs across OS")
	}
	code := execute([]string{"--config", "\x00"}, fakeLogger{}, func(path string) (config.Config, error) {
		return config.Config{}, nil
	}, fakeDownloader{})
	if code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestMainUsesInjectedDependencies(t *testing.T) {
	oldArgs := os.Args
	oldNewLogger := newLogger
	oldNewNetClient := newNetClient
	oldNewDownloader := newDownloader
	oldWarm := warmStationAreaCache
	oldLoad := loadConfigFn
	oldExit := exitFn
	defer func() {
		os.Args = oldArgs
		newLogger = oldNewLogger
		newNetClient = oldNewNetClient
		newDownloader = oldNewDownloader
		warmStationAreaCache = oldWarm
		loadConfigFn = oldLoad
		exitFn = oldExit
	}()

	os.Args = []string{"rajidou", "--config", "x.yaml"}
	newLogger = func() loggerAPI { return fakeLogger{} }
	newNetClient = func() *netx.Client { return netx.NewClient(0, netx.RetryOptions{Retries: 0}) }
	newDownloader = func(net *netx.Client) downloaderAPI { return fakeDownloader{} }
	warmed := false
	warmStationAreaCache = func(ctx context.Context, net *netx.Client) { warmed = true }
	loadConfigFn = func(path string) (config.Config, error) {
		return config.Config{
			Links:     []string{"a"},
			OutputDir: t.TempDir(),
			Jobs:      1,
		}, nil
	}
	gotExit := -1
	exitFn = func(code int) { gotExit = code }

	main()

	if !warmed {
		t.Fatal("expected warmStationAreaCache to be called")
	}
	if gotExit != 0 {
		t.Fatalf("want exit code 0, got %d", gotExit)
	}
}

func TestMainExitOnConfigLoadError(t *testing.T) {
	oldArgs := os.Args
	oldNewLogger := newLogger
	oldNewNetClient := newNetClient
	oldNewDownloader := newDownloader
	oldWarm := warmStationAreaCache
	oldLoad := loadConfigFn
	oldExit := exitFn
	defer func() {
		os.Args = oldArgs
		newLogger = oldNewLogger
		newNetClient = oldNewNetClient
		newDownloader = oldNewDownloader
		warmStationAreaCache = oldWarm
		loadConfigFn = oldLoad
		exitFn = oldExit
	}()

	os.Args = []string{"rajidou", "--config", "x.yaml"}
	newLogger = func() loggerAPI { return fakeLogger{} }
	newNetClient = func() *netx.Client { return netx.NewClient(0, netx.RetryOptions{Retries: 0}) }
	newDownloader = func(net *netx.Client) downloaderAPI { return fakeDownloader{} }
	warmStationAreaCache = func(ctx context.Context, net *netx.Client) {}
	loadConfigFn = func(path string) (config.Config, error) {
		return config.Config{}, fmt.Errorf("load fail")
	}
	gotExit := -1
	exitFn = func(code int) { gotExit = code }

	main()

	if gotExit != 1 {
		t.Fatalf("want exit code 1, got %d", gotExit)
	}
}
