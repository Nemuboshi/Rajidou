package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"rajidou/internal/cli"
	"rajidou/internal/config"
	"rajidou/internal/domain"
	"rajidou/internal/netx"
)

var (
	newLogger    = func() loggerAPI { return cli.Logger{} }
	newNetClient = func() *netx.Client {
		return netx.NewClient(45*time.Second, netx.RetryOptions{Retries: 3, BaseDelay: 300 * time.Millisecond, MaxDelay: 2 * time.Second})
	}
	newDownloader        = func(net *netx.Client) downloaderAPI { return domain.NewDownloader(net, 8) }
	warmStationAreaCache = func(ctx context.Context, net *netx.Client) { domain.WarmStationAreaCache(ctx, net) }
	loadConfigFn         = config.Load
	exitFn               = cli.Exit
)

type loggerAPI interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Success(msg string)
	Failure(msg string)
}

type downloaderAPI interface {
	ResolveToDetailURL(ctx context.Context, raw string) (string, error)
	DownloadFromDetailURL(ctx context.Context, detailURL string, opt domain.DownloadOptions) (string, error)
}

func execute(args []string, logger loggerAPI, cfgLoader func(path string) (config.Config, error), downloader downloaderAPI) int {
	cfgPath := cli.ParseArgs(args)
	resolvedCfg, err := filepath.Abs(cfgPath)
	if err != nil {
		logger.Error(formatError(err))
		return 1
	}
	cfg, err := cfgLoader(resolvedCfg)
	if err != nil {
		logger.Error(formatError(err))
		return 1
	}
	// Keep output paths deterministic for logs and downstream tooling.
	outputDir, _ := filepath.Abs(cfg.OutputDir)

	success := 0
	type failItem struct{ inputURL, reason string }
	fails := make([]failItem, 0)
	var mu sync.Mutex

	type task struct {
		index int
		url   string
	}
	jobs := cfg.Jobs
	// Bound worker count to a valid range so scheduling and channel lifecycles stay predictable.
	if jobs > len(cfg.Links) {
		jobs = len(cfg.Links)
	}
	if jobs < 1 {
		jobs = 1
	}
	taskCh := make(chan task)
	var wg sync.WaitGroup
	for w := 0; w < jobs; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range taskCh {
				progress := cli.NewDownloadProgress(fmt.Sprintf("segments[%d]", t.index+1))
				// Use a per-item timeout so one stalled URL does not block the whole run.
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				logger.Info("Input: " + t.url)
				detailURL, err := downloader.ResolveToDetailURL(ctx, t.url)
				if err != nil {
					// Release timer resources on all early returns.
					cancel()
					progress.Stop()
					msg := formatError(err)
					mu.Lock()
					fails = append(fails, failItem{inputURL: t.url, reason: msg})
					mu.Unlock()
					logger.Failure(t.url + " -> " + msg)
					continue
				}
				logger.Info("Resolved detail: " + detailURL)
				onProgress := func(done, total int) {
					progress.Update(done, total)
				}
				outPath, err := downloader.DownloadFromDetailURL(ctx, detailURL, domain.DownloadOptions{
					OutputDir:  outputDir,
					AreaID:     cfg.AreaID,
					OnProgress: onProgress,
				})
				cancel()
				progress.Stop()
				if err != nil {
					msg := formatError(err)
					mu.Lock()
					// Record failures instead of aborting so remaining inputs continue processing.
					fails = append(fails, failItem{inputURL: t.url, reason: msg})
					mu.Unlock()
					logger.Failure(t.url + " -> " + msg)
					continue
				}
				mu.Lock()
				success++
				mu.Unlock()
				logger.Success("Downloaded: " + outPath)
			}
		}()
	}
	for i, inputURL := range cfg.Links {
		taskCh <- task{index: i, url: inputURL}
	}
	close(taskCh)
	wg.Wait()

	logger.Info(fmt.Sprintf("Completed. success=%d failed=%d", success, len(fails)))
	if len(fails) > 0 {
		for _, f := range fails {
			logger.Warn("Failure detail: " + f.inputURL + " :: " + f.reason)
		}
		return 2
	}
	return 0
}

func main() {
	logger := newLogger()
	net := newNetClient()
	downloader := newDownloader(net)
	warmStationAreaCache(context.Background(), net)
	exitCode := execute(os.Args[1:], logger, loadConfigFn, downloader)
	exitFn(exitCode)
}

func formatError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
