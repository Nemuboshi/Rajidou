package domain

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"rajidou/internal/netx"
)

// Source map in this file:
// - ParseAACPackedHeaderSize: rajiko/modules/util.js (parseAAC)
// - worker download flow: rajiko/modules/timeshift.js
// - merge/write behavior: rajiko/modules/recording.js adaptation
// AudioDownloader fetches AAC segments concurrently and merges them in order.
type AudioDownloader struct {
	net         *netx.Client
	concurrency int
}

// NewAudioDownloader creates an AudioDownloader with bounded worker count.
// A non-positive concurrency value falls back to a practical default.
func NewAudioDownloader(net *netx.Client, concurrency int) *AudioDownloader {
	if concurrency <= 0 {
		concurrency = 8
	}
	return &AudioDownloader{net: net, concurrency: concurrency}
}

// ParseAACPackedHeaderSize returns the byte length of an ID3v2 header prefix.
// Radiko AAC chunks may start with metadata; callers skip this prefix so merged
// output is pure AAC payload.
func ParseAACPackedHeaderSize(data []byte) int {
	if len(data) < 10 {
		return 0
	}
	if data[0] != 73 || data[1] != 68 || data[2] != 51 {
		return 0
	}
	id3 := int(data[6])<<24 | int(data[7])<<16 | int(data[8])<<8 | int(data[9])
	return 10 + id3
}

var mergeBufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// DownloadAndMergeAacSegments downloads all segment URLs, strips optional ID3
// headers, and writes one merged AAC file. Segment order is preserved by index
// even when downloads complete out of order.
func (a *AudioDownloader) DownloadAndMergeAacSegments(ctx context.Context, urls []string, outputDir, fileName string, onProgress func(done, total int)) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	total := len(urls)
	onProgressSafe(onProgress, 0, total)

	type task struct {
		idx int
		url string
	}
	tasks := make(chan task)
	results := make([][]byte, total)
	errCh := make(chan error, 1)
	var once sync.Once
	var wg sync.WaitGroup
	var doneMu sync.Mutex
	done := 0

	worker := func() {
		defer wg.Done()
		for t := range tasks {
			_, b, err := a.net.GetBytes(ctx, t.url, nil)
			if err != nil {
				// Keep workers running so in-flight tasks can finish; report first error.
				once.Do(func() { errCh <- fmt.Errorf("segment fetch failed: %w", err) })
				continue
			}
			h := ParseAACPackedHeaderSize(b)
			if h > len(b) {
				h = 0
			}
			results[t.idx] = b[h:]
			doneMu.Lock()
			done++
			onProgressSafe(onProgress, done, total)
			doneMu.Unlock()
		}
	}

	n := a.concurrency
	if n > total && total > 0 {
		n = total
	}
	if n == 0 {
		n = 1
	}
	wg.Add(n)
	for i := 0; i < n; i++ {
		go worker()
	}
	for i, u := range urls {
		select {
		case <-ctx.Done():
			close(tasks)
			wg.Wait()
			return "", ctx.Err()
		default:
			tasks <- task{idx: i, url: u}
		}
	}
	close(tasks)
	wg.Wait()

	select {
	case err := <-errCh:
		return "", err
	default:
	}

	finalSize := 0
	for _, seg := range results {
		finalSize += len(seg)
	}
	buf := mergeBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Grow(finalSize)
	for _, seg := range results {
		_, _ = buf.Write(seg)
	}
	outPath := filepath.Join(outputDir, fileName)
	data := append([]byte(nil), buf.Bytes()...)
	mergeBufferPool.Put(buf)
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return "", err
	}
	abs, _ := filepath.Abs(outPath)
	return abs, nil
}

func onProgressSafe(fn func(done, total int), done, total int) {
	if fn != nil {
		fn(done, total)
	}
}
