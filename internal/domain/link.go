package domain

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"rajidou/internal/util"
)

// Source map in this file:
//   - URL classification/parsing helpers are CLI-specific,
//     compatible with Radiko timeshift URL format used by Rajiko.
const (
	searchMarker = "#!/search/timeshift"
	detailPrefix = "#!/ts/"
)

// LinkKind classifies Radiko links by workflow entry point.
type LinkKind string

const (
	// LinkKindSearch identifies a search result URL that must be resolved.
	LinkKindSearch LinkKind = "search"
	// LinkKindDetail identifies a direct timeshift detail URL.
	LinkKindDetail LinkKind = "detail"
	// LinkKindUnsupported identifies links outside supported timeshift flows.
	LinkKindUnsupported LinkKind = "unsupported"
)

// DetailRef is the normalized station/time identifier extracted from detail URLs.
type DetailRef struct {
	StationID string
	FT        string
}

// ClassifyRadikoLink returns which timeshift flow should handle the URL.
func ClassifyRadikoLink(raw string) LinkKind {
	if strings.Contains(raw, searchMarker) {
		return LinkKindSearch
	}
	if strings.Contains(raw, detailPrefix) {
		return LinkKindDetail
	}
	return LinkKindUnsupported
}

// ExtractDetailFromDetailURL parses a Radiko detail URL and validates its FT
// timestamp format to avoid propagating malformed identifiers downstream.
func ExtractDetailFromDetailURL(raw string) (DetailRef, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return DetailRef{}, fmt.Errorf("invalid detail URL: %s", raw)
	}
	hash := u.Fragment
	hash = strings.TrimPrefix(hash, "!")
	hash = strings.TrimPrefix(hash, "/")
	segs := strings.Split(hash, "/")
	if len(segs) < 3 || segs[0] != "ts" {
		return DetailRef{}, fmt.Errorf("invalid detail URL: %s", raw)
	}
	ft := segs[2]
	if _, err := util.ParseTimestamp(ft); err != nil {
		return DetailRef{}, fmt.Errorf("invalid ft timestamp in detail URL: %s", raw)
	}
	return DetailRef{StationID: segs[1], FT: ft}, nil
}

// PickLatestDetailURL selects the most recent detail URL that is not in the
// future relative to now, which matches expected timeshift availability.
func PickLatestDetailURL(urls []string, now time.Time) (string, error) {
	nowTS := util.FormatTimestamp(now)
	type item struct {
		url string
		ft  string
	}
	items := make([]item, 0, len(urls))
	for _, u := range urls {
		d, err := ExtractDetailFromDetailURL(u)
		if err != nil {
			// Ignore malformed entries and keep scanning other candidates.
			continue
		}
		if d.FT <= nowTS {
			items = append(items, item{url: u, ft: d.FT})
		}
	}
	if len(items) == 0 {
		return "", fmt.Errorf("no usable detail URL found in search results")
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ft > items[j].ft })
	return items[0].url, nil
}
