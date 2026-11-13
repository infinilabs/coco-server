/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package text_attachment_extraction

import (
	"strings"
)

// Page-level cleanup heuristics (W2 L0), both deliberately conservative —
// a wrong deletion loses real content, a missed deletion only costs noise.

const (
	// headerFooterMinPages: below this there is not enough evidence that a
	// repeating line is furniture rather than content.
	headerFooterMinPages = 5
	// headerFooterMinRatio: the line must sit at the same edge of at least
	// this share of pages (WeKnora's 60% discipline).
	headerFooterMinRatio = 0.6
	// headerFooterMaxRunes: long first/last lines are content, not furniture.
	headerFooterMaxRunes = 80

	// garbledRuneRatio: a page whose replacement-char share exceeds this is
	// flagged (Tika emits U+FFFD for bytes it cannot decode).
	garbledRuneRatio = 0.005
)

// stripRepeatingHeaderFooter removes first/last lines that repeat across
// >=60% of pages: only the very first and very last non-empty line of each
// page are candidates, they must be short, and digits normalize to '#'
// ("Page 3" == "Page 4"). Pages without the furniture keep their lines.
func stripRepeatingHeaderFooter(pages []string) []string {
	if len(pages) < headerFooterMinPages {
		return pages
	}

	type edge struct{ idx, line int } // page index, line index within the page
	firstSeen, lastSeen := map[string][]edge{}, map[string][]edge{}
	pageLines := make([][]string, len(pages))
	for i, p := range pages {
		lines := strings.Split(p, "\n")
		pageLines[i] = lines
		first, last := -1, -1
		for j, l := range lines {
			if strings.TrimSpace(l) == "" {
				continue
			}
			if first == -1 {
				first = j
			}
			last = j
		}
		if first != -1 && runeLen(lines[first]) <= headerFooterMaxRunes {
			k := normalizeHeaderFooterLine(lines[first])
			if k != "" {
				firstSeen[k] = append(firstSeen[k], edge{i, first})
			}
		}
		if last != -1 && last != first && runeLen(lines[last]) <= headerFooterMaxRunes {
			k := normalizeHeaderFooterLine(lines[last])
			if k != "" {
				lastSeen[k] = append(lastSeen[k], edge{i, last})
			}
		}
	}

	remove := map[edge]bool{}
	consider := func(seen map[string][]edge) {
		for _, edges := range seen {
			if float64(len(edges))/float64(len(pages)) >= headerFooterMinRatio {
				for _, e := range edges {
					remove[e] = true
				}
			}
		}
	}
	consider(firstSeen)
	consider(lastSeen)
	if len(remove) == 0 {
		return pages
	}

	out := make([]string, len(pages))
	for i := range pageLines {
		var b strings.Builder
		for j, l := range pageLines[i] {
			if remove[edge{i, j}] {
				continue
			}
			if b.Len() > 0 || strings.TrimSpace(l) != "" {
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
				b.WriteString(l)
			}
		}
		out[i] = b.String()
	}
	return out
}

// normalizeHeaderFooterLine prepares a header/footer candidate line for
// cross-page comparison: page numbers and dates vary per page, so digits
// collapse to a placeholder before counting occurrences.
func normalizeHeaderFooterLine(line string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(strings.ToLower(line)) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteByte('#')
		case r == ' ' || r == '\t':
			// collapse runs of whitespace
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// detectGarbledPages returns the 1-based page numbers whose replacement-char
// share exceeds the threshold — broken encodings and mis-detected binaries.
// Detection only marks (metadata + log): rerouting to forced OCR is an L1
// backend concern, not something this zero-model stage can fix.
func detectGarbledPages(pages []string) []int {
	var garbled []int
	for i, p := range pages {
		if p == "" {
			continue
		}
		runes := []rune(p)
		bad := 0
		for _, r := range runes {
			if r == '�' {
				bad++
			}
		}
		if float64(bad)/float64(len(runes)) > garbledRuneRatio {
			garbled = append(garbled, i+1)
		}
	}
	return garbled
}

func runeLen(s string) int {
	return len([]rune(s))
}
