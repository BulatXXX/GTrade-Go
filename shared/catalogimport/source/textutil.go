package source

import (
	"html"
	"regexp"
	"strings"
)

var (
	htmlBlockBreakRE = regexp.MustCompile(`(?i)<\s*/?(br|p|div|li|h[1-6])[^>]*>`)
	htmlTagRE        = regexp.MustCompile(`<[^>]+>`)
	htmlMultiSpaceRE = regexp.MustCompile(`[ \t]+`)
	htmlMultiNewline = regexp.MustCompile(`\n{3,}`)
)

// stripHTML removes HTML markup from EVE-style rich text and returns plain text
// with normalized whitespace. Block-level tags (br, p, div, li, hN) are turned
// into newlines so paragraph structure survives; remaining tags are dropped and
// HTML entities are decoded.
func stripHTML(s string) string {
	if s == "" {
		return s
	}

	out := htmlBlockBreakRE.ReplaceAllString(s, "\n")
	out = htmlTagRE.ReplaceAllString(out, "")
	out = html.UnescapeString(out)
	out = strings.ReplaceAll(out, " ", " ")
	out = htmlMultiSpaceRE.ReplaceAllString(out, " ")
	out = htmlMultiNewline.ReplaceAllString(out, "\n\n")

	lines := strings.Split(out, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
