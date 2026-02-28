package domain

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"rajidou/internal/netx"
	"rajidou/internal/util"
)

// Source map in this file:
// - playlistCreateURL behavior: rajiko/modules/timeshift.js (playlist_create_url)
// - seek/chunklist loop: rajiko/modules/timeshift.js (downloadtimeShift)
// PlaylistBuilder expands timeshift metadata into ordered AAC segment URLs.
type PlaylistBuilder struct {
	net *netx.Client
}

// NewPlaylistBuilder creates a playlist builder backed by the shared HTTP client.
func NewPlaylistBuilder(net *netx.Client) *PlaylistBuilder {
	return &PlaylistBuilder{net: net}
}

func (p *PlaylistBuilder) playlistCreateURL(ctx context.Context, stationID string) (string, error) {
	url := fmt.Sprintf("https://radiko.jp/v3/station/stream/pc_html5/%s.xml", stationID)
	status, xml, err := p.net.GetText(ctx, url, nil)
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("station stream xml failed: %d", status)
	}
	re := regexp.MustCompile(`<playlist_create_url>(.*?)</playlist_create_url>`)
	m := re.FindStringSubmatch(xml)
	if len(m) >= 2 {
		return strings.TrimSpace(m[1]), nil
	}
	// Older stations may omit this field; use the known default endpoint.
	return "https://tf-f-rpaa-radiko.smartstream.ne.jp/tf/playlist.m3u8", nil
}

// SegmentInput contains the server-side parameters required to request
// timeshift playlists and authorized chunklists.
type SegmentInput struct {
	StationID string
	FT        string
	TO        string
	Token     string
	AreaID    string
}

// BuildSegmentURLs iterates seek windows between FT and TO and collects media
// segment URLs from each chunklist response.
func (p *PlaylistBuilder) BuildSegmentURLs(ctx context.Context, in SegmentInput) ([]string, error) {
	const fixedSeek = 300
	base, err := p.playlistCreateURL(ctx, in.StationID)
	if err != nil {
		return nil, err
	}

	links := make([]string, 0, 1024)
	seek := in.FT
	endDate, err := util.ParseTimestamp(in.TO)
	if err != nil {
		return nil, err
	}

	for {
		seekDate, err := util.ParseTimestamp(seek)
		if err != nil {
			return nil, err
		}
		if !seekDate.Before(endDate) {
			break
		}

		playlistURL := fmt.Sprintf("%s?lsid=%s&station_id=%s&l=%d&start_at=%s&end_at=%s&type=b&ft=%s&to=%s&seek=%s",
			base, randomHex(16), in.StationID, fixedSeek, in.FT, in.TO, in.FT, in.TO, seek,
		)
		status, playlistText, err := p.net.GetText(ctx, playlistURL, map[string]string{
			"X-Radiko-AreaId":    in.AreaID,
			"X-Radiko-AuthToken": in.Token,
		})
		if err != nil {
			return nil, err
		}
		if status == 403 || strings.TrimSpace(playlistText) == "expired" || status < 200 || status >= 300 {
			// 403/expired/non-2xx all indicate the current seek window is unusable.
			return nil, fmt.Errorf("playlist request failed at seek=%s: %d", seek, status)
		}

		detailURL, err := firstDataLine(playlistText)
		if err != nil {
			return nil, err
		}
		chunkStatus, chunkText, err := p.net.GetText(ctx, detailURL, nil)
		if err != nil {
			return nil, err
		}
		if chunkStatus < 200 || chunkStatus >= 300 {
			return nil, fmt.Errorf("chunklist request failed: %d", chunkStatus)
		}
		links = append(links, allDataLines(chunkText)...)
		next, err := util.StepTimestamp(seek, fixedSeek)
		if err != nil {
			return nil, err
		}
		seek = next
	}

	return links, nil
}

func firstDataLine(m3u8 string) (string, error) {
	lines := allDataLines(m3u8)
	if len(lines) == 0 {
		return "", fmt.Errorf("m3u8 has no media lines")
	}
	return lines[0], nil
}

func allDataLines(m3u8 string) []string {
	out := make([]string, 0, strings.Count(m3u8, "\n")+1)
	start := 0
	for i := 0; i <= len(m3u8); i++ {
		if i < len(m3u8) && m3u8[i] != '\n' {
			continue
		}
		line := m3u8[start:i]
		start = i + 1
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}
