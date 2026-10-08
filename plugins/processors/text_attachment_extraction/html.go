/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package text_attachment_extraction

import (
	"fmt"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// htmlToText converts an (X)HTML document to plain text with light structure
// markers (W2 L0.5): h1-h6 become markdown headings so the chunker and the
// future breadcrumb (W3) can key on them; table rows stay on one line with
// cells joined by " | "; script/style/noscript are dropped.
func htmlToText(r io.Reader) (string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return "", err
	}
	return selectionToStructuredText(doc.Selection), nil
}

// selectionToStructuredText renders a goquery selection with structure
// markers. Shared by the standalone HTML/EPUB/MHTML routes and the Tika
// XHTML page path (appendPage) — before this, Tika pages went through
// plain .Text() and lost every heading and table boundary, which is
// exactly the "big tables smear into positional soup" failure mode.
func selectionToStructuredText(s *goquery.Selection) string {
	if s == nil {
		return ""
	}
	s.Find("script,style,noscript,iframe").Remove()

	var b strings.Builder

	// heading levels keep their hierarchy as markdown — the cheapest
	// structure signal a zero-model parser can preserve
	s.Find("h1,h2,h3,h4,h5,h6").Each(func(_ int, h *goquery.Selection) {
		level := 1
		switch goquery.NodeName(h) {
		case "h2":
			level = 2
		case "h3":
			level = 3
		case "h4":
			level = 4
		case "h5":
			level = 5
		case "h6":
			level = 6
		}
		if t := strings.TrimSpace(h.Text()); t != "" {
			fmt.Fprintf(&b, "%s %s\n", strings.Repeat("#", level), t)
		}
	})

	s.Find("table").Each(func(_ int, table *goquery.Selection) {
		table.Find("tr").Each(func(_ int, tr *goquery.Selection) {
			var cells []string
			tr.Find("td,th").Each(func(_ int, cell *goquery.Selection) {
				if t := strings.Join(strings.Fields(cell.Text()), " "); t != "" {
					cells = append(cells, t)
				}
			})
			if len(cells) > 0 {
				b.WriteString(strings.Join(cells, " | "))
				b.WriteByte('\n')
			}
		})
	})

	// paragraphs and list items carry the body text
	s.Find("p,li,blockquote,pre").Each(func(_ int, para *goquery.Selection) {
		if t := strings.Join(strings.Fields(para.Text()), " "); t != "" {
			b.WriteString(t)
			b.WriteByte('\n')
		}
	})

	out := b.String()
	if strings.TrimSpace(out) == "" {
		// structure-less markup: fall back to the full text body — the
		// heading/table/paragraph selectors found nothing worth marking
		out = s.Text()
	}
	return strings.TrimRight(strings.TrimSpace(out), "\n")
}
