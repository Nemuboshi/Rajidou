package util

import "strings"

// Source map in this file:
// - output naming/sanitization is CLI-specific but matches Rajidou TS behavior.
// BuildProgramFileName returns "<sanitized title> - <YYYYMMDD>.aac".
//
// It derives the date from the first 8 characters of ft when available; if ft
// is shorter, the full ft string is used unchanged.
func BuildProgramFileName(title, ft string) string {
	date := ft
	if len(ft) >= 8 {
		date = ft[:8]
	}
	safeTitle := SanitizeFileNamePart(strings.TrimSpace(title))
	return safeTitle + " - " + date + ".aac"
}

// SanitizeFileNamePart removes characters invalid on common filesystems,
// collapses repeated separators/whitespace, and returns "program" for empty
// results.
func SanitizeFileNamePart(value string) string {
	replacer := strings.NewReplacer(
		"\\", "_",
		"/", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	s := replacer.Replace(value)
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return "program"
	}
	return s
}
