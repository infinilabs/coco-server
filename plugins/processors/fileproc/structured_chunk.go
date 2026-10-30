/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package fileproc

import (
	"regexp"
	"strings"

	"infini.sh/coco/core"
)

// Structure-aware chunking (W3/D10): the XHTML structure route (W2) already
// renders pages with markdown heading markers — this splitter consumes
// them. Sections split at headings, each chunk carries the accumulated
// heading breadcrumb, chunks respect a hard rune cap with a bounded
// overlap, and a section's chunks reassemble to the section text
// losslessly (overlap prefixes excluded).
//
// Legacy behavior stays available in SplitPagesToChunks; documents parsed
// before the structure route simply contain no heading markers, and every
// line lands in one implicit root section — windowed with the same budget
// and overlap, so even unstructured input chunks better than the old
// mid-word guillotine.

// structuredOverlapFraction of the chunk budget carried as the next
// chunk's prefix, 15% — the design's 15-20% band, low end.
const structuredOverlapFraction = 15 // percent

type StructuredChunkConfig struct {
	// ChunkSize is the hard rune cap per chunk.
	ChunkSize int
	// OverlapRunes is the bounded prefix carried into the next chunk
	// within the same section; derived from ChunkSize when zero.
	OverlapRunes int
}

// SplitStructuredPages splits heading-marked page texts into breadcrumbed
// chunks. Chunks never span a heading boundary (a heading opens a new
// section), page ranges are tracked, and single oversized lines are
// hard-split at the cap — the one place losslessness gives way to the
// hard-cap contract.
func SplitStructuredPages(pages []string, cfg StructuredChunkConfig) []core.DocumentChunk {
	if cfg.ChunkSize <= 0 {
		return nil
	}
	if cfg.OverlapRunes <= 0 {
		cfg.OverlapRunes = cfg.ChunkSize * structuredOverlapFraction / 100
	}
	if cfg.OverlapRunes >= cfg.ChunkSize {
		cfg.OverlapRunes = cfg.ChunkSize / 4
	}

	s := &structuredSplitter{cfg: cfg, breadcrumb: []string{}}
	for pageNo, page := range pages {
		s.page = pageNo + 1
		for _, line := range strings.Split(page, "\n") {
			s.line(line)
		}
	}
	s.flush()

	out := make([]core.DocumentChunk, 0, len(s.chunks))
	for _, c := range s.chunks {
		out = append(out, core.DocumentChunk{
			Range:      core.ChunkRange{Start: c.startPage, End: c.endPage},
			Text:       c.text,
			Breadcrumb: c.breadcrumb,
		})
	}
	return out
}

type structuredChunk struct {
	text       string
	breadcrumb string
	startPage  int
	endPage    int
}

type structuredSplitter struct {
	cfg        StructuredChunkConfig
	breadcrumb []string
	page       int

	section           strings.Builder // current section's text
	sectionStartPage  int
	sectionEndPage    int
	sectionHeadingTxt string // the opening heading line, verbatim — an
	// "orphan heading" (heading immediately followed by a heading) emits
	// no chunk: it carries no retrievable content

	chunks []structuredChunk
}

var headingLine = regexp.MustCompile(`^(#{1,6}) (.*)$`)

func (s *structuredSplitter) line(line string) {
	trimmed := strings.TrimRight(line, "\r")
	if m := headingLine.FindStringSubmatch(trimmed); m != nil {
		s.flush()
		level := len(m[1])
		title := strings.TrimSpace(m[2])
		s.breadcrumb = s.breadcrumb[:min(len(s.breadcrumb), level-1)]
		s.breadcrumb = append(s.breadcrumb, title)
		s.sectionStartPage = s.page
		s.sectionHeadingTxt = trimmed
	}

	if s.sectionStartPage == 0 {
		s.sectionStartPage = s.page
	}
	s.sectionEndPage = s.page

	if strings.TrimSpace(trimmed) == "" {
		// keep paragraph structure inside the section but collapse the
		// leading blank after a heading flush
		if s.section.Len() == 0 {
			return
		}
		s.section.WriteByte('\n')
		return
	}
	s.section.WriteString(trimmed)
	s.section.WriteByte('\n')
}

func (s *structuredSplitter) flush() {
	text := strings.TrimRight(s.section.String(), "\n")
	heading := s.sectionHeadingTxt
	s.section.Reset()
	s.sectionHeadingTxt = ""
	if text == "" || text == heading {
		// empty section or an orphan heading — nothing retrievable
		s.sectionStartPage, s.sectionEndPage = 0, 0
		return
	}
	breadcrumb := strings.Join(s.breadcrumb, " > ")

	// window the section text into capped chunks with bounded overlap;
	// lines are the preferred cut points, the rune cap is the hard one
	rest := []rune(text)
	var overlap []rune
	for len(rest) > 0 {
		prefix := overlap
		if len(prefix) >= s.cfg.ChunkSize {
			prefix = nil
		}
		buf, remaining := appendRunes(prefix, rest, s.cfg.ChunkSize)
		if len(remaining) == len(rest) {
			// no progress with the overlap prefix — drop it and force
			// at least one rune so the loop always advances
			buf, remaining = appendRunes(nil, rest, s.cfg.ChunkSize)
		}
		chunkText := strings.TrimRight(string(buf), "\n")
		if chunkText != "" {
			s.chunks = append(s.chunks, structuredChunk{
				text:       chunkText,
				breadcrumb: breadcrumb,
				startPage:  s.sectionStartPage,
				endPage:    s.sectionEndPage,
			})
		}
		overlap = tailRunes(buf, s.cfg.OverlapRunes)
		rest = remaining
	}

	s.sectionStartPage, s.sectionEndPage = 0, 0
}

// appendRunes packs runes from rest into prefix without exceeding cap,
// preferring a line boundary in the final quarter of the room. Returns the
// packed buffer and the unconsumed runes. When even prefix alone exceeds
// the cap the hard-cap contract wins: prefix is truncated to the cap.
func appendRunes(prefix, rest []rune, cap int) ([]rune, []rune) {
	buf := append([]rune{}, prefix...)
	if len(buf) >= cap {
		return buf[:cap], rest
	}
	room := cap - len(buf)
	if len(rest) <= room {
		return append(buf, rest...), nil
	}
	take := room
	// prefer cutting at a line boundary within the last quarter of room
	if idx := lastIndexOf(rest[:take], '\n'); idx >= 0 && idx >= take-cap/4 && idx > 0 {
		take = idx + 1
	}
	return append(buf, rest[:take]...), rest[take:]
}

func lastIndexOf(v []rune, r rune) int {
	last := -1
	for i, c := range v {
		if c == r {
			last = i
		}
	}
	return last
}

// tailRunes returns up to n trailing runes, cut to a line start when one
// sits nearby so an overlap never begins mid-line.
func tailRunes(v []rune, n int) []rune {
	if n <= 0 || len(v) == 0 {
		return nil
	}
	if n > len(v) {
		n = len(v)
	}
	tail := v[len(v)-n:]
	if idx := indexRune(tail, '\n'); idx >= 0 {
		tail = tail[idx+1:]
	}
	return tail
}

func indexRune(v []rune, r rune) int {
	for i, c := range v {
		if c == r {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
