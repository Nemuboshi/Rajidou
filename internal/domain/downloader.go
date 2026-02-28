package domain

import (
	"context"
	"fmt"

	"rajidou/internal/netx"
	"rajidou/internal/util"
)

// Source map in this file:
// - end-to-end timeshift download orchestration: rajiko/modules/timeshift.js
// - Node/extension-specific APIs are replaced by Go CLI abstractions.
// DownloadOptions configures a full timeshift download request.
type DownloadOptions struct {
	OutputDir  string
	AreaID     string
	OnProgress func(done, total int)
}

type resolverAPI interface {
	ResolveToDetailURL(ctx context.Context, raw string) (string, error)
}

type authAPI interface {
	RetrieveToken(ctx context.Context, areaID string) (string, error)
}

type programAPI interface {
	ResolveProgramMeta(ctx context.Context, stationID, ft string) (ProgramMeta, error)
}

type playlistAPI interface {
	BuildSegmentURLs(ctx context.Context, in SegmentInput) ([]string, error)
}

type audioAPI interface {
	DownloadAndMergeAacSegments(ctx context.Context, urls []string, outputDir, fileName string, onProgress func(done, total int)) (string, error)
}

// Downloader orchestrates resolution, auth, playlist expansion, and audio merge.
type Downloader struct {
	resolver      resolverAPI
	resolveAreaID func(ctx context.Context, stationID string) (string, error)
	program       programAPI
	playlist      playlistAPI
	auth          authAPI
	audio         audioAPI
}

// NewDownloader wires the domain workflow with default concrete components.
func NewDownloader(net *netx.Client, concurrency int) *Downloader {
	return &Downloader{
		resolver: NewPageResolver(net),
		resolveAreaID: func(ctx context.Context, stationID string) (string, error) {
			return ResolveStationAreaID(ctx, net, stationID)
		},
		program:  NewProgramResolver(net),
		playlist: NewPlaylistBuilder(net),
		auth:     NewAuthClient(net),
		audio:    NewAudioDownloader(net, concurrency),
	}
}

// ResolveToDetailURL normalizes an input link into a Radiko detail URL.
func (d *Downloader) ResolveToDetailURL(ctx context.Context, raw string) (string, error) {
	return d.resolver.ResolveToDetailURL(ctx, raw)
}

// DownloadFromDetailURL executes the full timeshift workflow from a detail URL.
// If AreaID is not provided, it is resolved from station metadata before auth.
func (d *Downloader) DownloadFromDetailURL(ctx context.Context, detailURL string, opt DownloadOptions) (string, error) {
	detail, err := ExtractDetailFromDetailURL(detailURL)
	if err != nil {
		return "", err
	}
	areaID := opt.AreaID
	if areaID == "" {
		areaID, err = d.resolveAreaID(ctx, detail.StationID)
		if err != nil {
			return "", err
		}
	}
	token, err := d.auth.RetrieveToken(ctx, areaID)
	if err != nil {
		return "", err
	}
	meta, err := d.program.ResolveProgramMeta(ctx, detail.StationID, detail.FT)
	if err != nil {
		return "", err
	}
	segmentURLs, err := d.playlist.BuildSegmentURLs(ctx, SegmentInput{
		StationID: detail.StationID,
		FT:        meta.FT,
		TO:        meta.TO,
		Token:     token,
		AreaID:    areaID,
	})
	if err != nil {
		return "", err
	}
	if len(segmentURLs) == 0 {
		return "", fmt.Errorf("no segments found")
	}
	fileName := util.BuildProgramFileName(meta.Title, meta.FT)
	return d.audio.DownloadAndMergeAacSegments(ctx, segmentURLs, opt.OutputDir, fileName, opt.OnProgress)
}
