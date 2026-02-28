package domain

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"rajidou/internal/netx"
)

// Source map in this file:
//   - weekly program range/title resolution behavior adapted from
//     rajiko/modules/timeshift.js data lookup logic.
//
// ProgramResolver resolves program metadata from Radiko weekly XML feeds.
type ProgramResolver struct {
	net *netx.Client
}

// ProgramMeta describes the time window and title required for download naming
// and playlist range requests.
type ProgramMeta struct {
	FT    string
	TO    string
	Title string
}

// NewProgramResolver creates a ProgramResolver backed by the shared HTTP client.
func NewProgramResolver(net *netx.Client) *ProgramResolver {
	return &ProgramResolver{net: net}
}

// ResolveProgramMeta fetches weekly XML and extracts TO/title for the exact FT
// program block identified by station and start timestamp.
func (r *ProgramResolver) ResolveProgramMeta(ctx context.Context, stationID, ft string) (ProgramMeta, error) {
	url := fmt.Sprintf("https://api.radiko.jp/program/v3/weekly/%s.xml", stationID)
	status, xml, err := r.net.GetText(ctx, url, nil)
	if err != nil {
		return ProgramMeta{}, err
	}
	if status < 200 || status >= 300 {
		return ProgramMeta{}, fmt.Errorf("weekly program xml failed: %d", status)
	}

	esc := regexp.QuoteMeta(ft)
	re := regexp.MustCompile(`<prog\s+[^>]*ft="` + esc + `"\s+to="(\d{14})"[^>]*>([\s\S]*?)</prog>`)
	m := re.FindStringSubmatch(xml)
	if len(m) < 3 {
		return ProgramMeta{}, fmt.Errorf("cannot find program range for station=%s ft=%s", stationID, ft)
	}
	block := m[2]
	title := ""
	tm := regexp.MustCompile(`<title>([\s\S]*?)</title>`).FindStringSubmatch(block)
	if len(tm) >= 2 {
		title = decodeXML(strings.TrimSpace(tm[1]))
	}
	return ProgramMeta{FT: ft, TO: m[1], Title: title}, nil
}

func decodeXML(s string) string {
	r := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
	)
	return r.Replace(s)
}
