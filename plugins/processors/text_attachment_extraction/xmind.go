/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package text_attachment_extraction

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"strings"
)

// Native XMind parsing (W2 L0.5): Tika cannot read XMind at all — this is
// the one format with no fallback backend. content.json (XMind 8+/Zen)
// carries sheets of topic trees; every topic has the same shape (title,
// notes, children.attached), so the recursion is direct. Each sheet becomes
// one page as an indented markdown outline, notes as quote lines.

type xmindNotes struct {
	Plain struct {
		Content string `json:"content"`
	} `json:"plain"`
}

type xmindTopic struct {
	Title    string `json:"title"`
	Notes    *xmindNotes
	Children struct {
		Attached []xmindTopic `json:"attached"`
	} `json:"children"`
}

type xmindSheet struct {
	Title string     `json:"title"`
	Root  xmindTopic `json:"rootTopic"`
}

func parseXmind(path string) ([]string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("not a readable xmind (zip): %w", err)
	}
	defer zr.Close()

	var contentFile *zip.File
	for _, f := range zr.File {
		if f.Name == "content.json" {
			contentFile = f
			break
		}
	}
	if contentFile == nil {
		return nil, fmt.Errorf("xmind content.json missing (legacy content.xml format unsupported)")
	}
	rc, err := contentFile.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var sheets []xmindSheet
	if err := json.NewDecoder(rc).Decode(&sheets); err != nil {
		return nil, fmt.Errorf("xmind content.json unreadable: %w", err)
	}

	pages := make([]string, 0, len(sheets))
	for i, sheet := range sheets {
		var b strings.Builder
		name := strings.TrimSpace(sheet.Title)
		if name == "" {
			name = fmt.Sprintf("Sheet %d", i+1)
		}
		fmt.Fprintf(&b, "# %s\n", name)
		writeXmindTopic(&b, sheet.Root, 0)
		page := strings.TrimRight(b.String(), "\n")
		if len(page) > len(name)+2 { // more than just the sheet heading
			pages = append(pages, page)
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("xmind has no renderable topics")
	}
	return pages, nil
}

func writeXmindTopic(b *strings.Builder, t xmindTopic, depth int) {
	if title := strings.TrimSpace(t.Title); title != "" {
		b.WriteString(strings.Repeat("  ", depth) + "- " + title + "\n")
	}
	if t.Notes != nil {
		if note := strings.TrimSpace(t.Notes.Plain.Content); note != "" {
			for _, line := range strings.Split(note, "\n") {
				if line = strings.TrimSpace(line); line != "" {
					b.WriteString(strings.Repeat("  ", depth+1) + "> " + line + "\n")
				}
			}
		}
	}
	for _, child := range t.Children.Attached {
		writeXmindTopic(b, child, depth+1)
	}
}
