package domain

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"rajidou/internal/netx"
)

// Source map in this file:
// - search/detail URL handling inspired by rajiko/pages/popup.js.
// - search API adaptation for CLI is implemented here.
// PageResolver resolves user-provided links into concrete detail URLs.
type PageResolver struct {
	net *netx.Client
}

// NewPageResolver creates a resolver backed by the shared HTTP client.
func NewPageResolver(net *netx.Client) *PageResolver {
	return &PageResolver{net: net}
}

// ResolveToDetailURL accepts search or detail links and returns a detail URL.
// Search links are expanded through the Radiko search API and then reduced to
// a single best candidate.
func (r *PageResolver) ResolveToDetailURL(ctx context.Context, raw string) (string, error) {
	kind := ClassifyRadikoLink(raw)
	if kind == LinkKindDetail {
		return raw, nil
	}
	if kind != LinkKindSearch {
		return "", fmt.Errorf("unsupported link: %s", raw)
	}
	links, err := r.fetchDetailLinksFromSearchAPI(ctx, raw)
	if err != nil {
		return "", err
	}
	if len(links) == 0 {
		return "", fmt.Errorf("no detail links found in search page: %s", raw)
	}
	return PickLatestDetailURL(links, time.Now())
}

// BuildDetailURLsFromSearchAPIData converts search API JSON payloads into
// canonical Radiko detail URLs and drops incomplete/invalid records.
func BuildDetailURLsFromSearchAPIData(payload []byte) ([]string, error) {
	var root struct {
		Data []struct {
			StationID string `json:"station_id"`
			StartTime string `json:"start_time"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(root.Data))
	for _, item := range root.Data {
		if item.StationID == "" || item.StartTime == "" {
			continue
		}
		ft := toFTTimestamp(item.StartTime)
		if ft == "" {
			continue
		}
		urls = append(urls, fmt.Sprintf("https://radiko.jp/#!/ts/%s/%s", item.StationID, ft))
	}
	return urls, nil
}

func (r *PageResolver) fetchDetailLinksFromSearchAPI(ctx context.Context, raw string) ([]string, error) {
	key := extractSearchKeyFromURL(raw)
	if key == "" {
		return nil, nil
	}
	u, _ := url.Parse("https://api.annex-cf.radiko.jp/v1/programs/legacy/perl/program/search")
	q := u.Query()
	q.Set("key", key)
	q.Set("filter", "")
	q.Set("start_day", "")
	q.Set("end_day", "")
	q.Set("area_id", "")
	q.Set("cur_area_id", "")
	q.Set("uid", randomHex(16))
	q.Set("row_limit", "12")
	q.Set("app_id", "pc")
	q.Set("action_id", "0")
	u.RawQuery = q.Encode()

	status, body, err := r.net.GetBytes(ctx, u.String(), nil)
	if err != nil {
		// Keep search resolution tolerant: callers receive "no links" instead of
		// transport-layer noise for this optional expansion path.
		return nil, nil
	}
	if status < 200 || status >= 300 {
		// Non-2xx search responses are treated as empty results by design.
		return nil, nil
	}
	return BuildDetailURLsFromSearchAPIData(body)
}

func extractSearchKeyFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	h := u.Fragment
	if len(h) > 1 && h[0] == '!' {
		h = h[1:]
	}
	idx := -1
	for i := 0; i < len(h); i++ {
		if h[i] == '?' {
			idx = i
			break
		}
	}
	if idx < 0 || idx+1 >= len(h) {
		return ""
	}
	q, _ := url.ParseQuery(h[idx+1:])
	return q.Get("key")
}

var startTimePattern = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2}):(\d{2})$`)

func toFTTimestamp(start string) string {
	m := startTimePattern.FindStringSubmatch(start)
	if len(m) != 7 {
		return ""
	}
	return m[1] + m[2] + m[3] + m[4] + m[5] + m[6]
}

func randomHex(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
