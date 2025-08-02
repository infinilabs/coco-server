/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package confluence

import (
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
)

func convertFromWiki(src *Content, doc *common.Document, baseURL *url.URL) {
	if src.Body.View != nil {
		textContent, err := htmlToText(src.Body.View.Value)
		if err != nil {
			_ = log.Warnf("[confluence connector] failed to convert html to text for src [id=%s]: %v", src.ID, err)

			// fallback to original content value
			doc.Content = src.Body.View.Value
		} else {
			// use parsed content
			doc.Content = textContent
		}
		// content size
		doc.Size = len(doc.Content)
	}

	if src.Links != nil && src.Links.WebUI != "" {
		webUiURL, _ := url.Parse(baseURL.Path + src.Links.WebUI)
		doc.URL = baseURL.ResolveReference(webUiURL).String()
	}

	if src.History != nil {
		if t, err := time.Parse(time.RFC3339Nano, src.History.CreatedDate); err == nil {
			doc.Created = &t
		}
		if src.History.CreatedBy.DisplayName != "" {
			doc.Owner = &common.UserInfo{
				UserName: src.History.CreatedBy.DisplayName,
				UserID:   src.History.CreatedBy.UserKey,
			}
		}
	}
	if src.Version != nil {
		if t, err := time.Parse(time.RFC3339Nano, src.Version.When); err == nil {
			doc.Updated = &t
		}
		if src.Version.By != nil && src.Version.By.DisplayName != "" {
			doc.Owner = &common.UserInfo{
				UserName: src.Version.By.DisplayName,
				UserID:   src.Version.By.UserKey,
			}
		}
	}
	if doc.Updated == nil {
		doc.Updated = doc.Created
	}
	doc.Metadata["type"] = src.Type
}

func convertFromAttachment(src *Content, doc *common.Document, baseURL *url.URL) {
	doc.Content = "" // Attachment src is not indexed directly

	// for attachment get size from `Extensions`
	if src.Extensions != nil {
		doc.Size = src.Extensions.FileSize
		doc.Metadata["media_type"] = src.Extensions.MediaType
	}

	if src.Links != nil && src.Links.Download != "" {
		downloadURL, _ := url.Parse(baseURL.Path + src.Links.Download)
		doc.URL = baseURL.ResolveReference(downloadURL).String()
	}

	if src.History != nil {
		if t, err := time.Parse(time.RFC3339Nano, src.History.CreatedDate); err == nil {
			doc.Created = &t
			doc.Updated = &t // For attachments, updated time is same as created time
		}
		if src.History.CreatedBy.DisplayName != "" {
			doc.Owner = &common.UserInfo{
				UserName: src.History.CreatedBy.DisplayName,
				UserID:   src.History.CreatedBy.AccountID,
			}
		}
	}
	doc.Metadata["type"] = src.Type
}

// htmlToText converts Confluence HTML content to plain text.
// It is optimized to handle Confluence-specific structures like macros and tables
// to improve search accuracy.
func htmlToText(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.Grow(len(html))

	// A map of tags that should have a space appended after their content is processed.
	// This is crucial for separating content from different blocks, like table cells or list items.
	separatorTags := map[string]bool{
		"p": true, "div": true, "li": true, "br": true, "hr": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"td": true, "th": true, "tr": true, "blockquote": true, "article": true,
		"section": true, "main": true, "header": true, "footer": true,
	}

	var traverse func(*goquery.Selection)
	traverse = func(s *goquery.Selection) {
		s.Each(func(_ int, node *goquery.Selection) {
			// If it's a text node, append its text.
			if goquery.NodeName(node) == "#text" {
				sb.WriteString(node.Text())
				return
			}

			// Ignore script/style tags as they don't contain searchable content.
			if node.Is("script, style") {
				return
			}

			// --- Confluence-Specific Macro Handling ---

			// Handle Draw.io diagrams. The content is usually an image or complex script data,
			// which is not useful for full-text search. We skip it entirely.
			if node.Is("div[id^='drawio-macro-data-']") {
				return // Skip this node and all its children
			}

			// Handle Info, Note, Warning, and Tip macros by prepending a semantic label.
			// This adds context for search, e.g., "Note: some important text".
			if node.Is("div.confluence-information-macro") {
				var macroType string
				if node.HasClass("confluence-information-macro-note") {
					macroType = "Note"
				} else if node.HasClass("confluence-information-macro-warning") {
					macroType = "Warning"
				} else if node.HasClass("confluence-information-macro-info") {
					macroType = "Info"
				} else if node.HasClass("confluence-information-macro-tip") {
					macroType = "Tip"
				}
				if macroType != "" {
					// Prepend the label and a space, then allow traversal to continue into the macro body.
					sb.WriteString(macroType + ": ")
				}
			}

			// Handle Code Block macros. We extract the text content at once and prevent further recursion
			// to ensure correct formatting and avoid mangling the code.
			if node.Is("div.code.panel") {
				codeText := node.Find("pre").Text()
				// Replace newlines with spaces to treat the code as a single block of text for search.
				normalizedCode := strings.ReplaceAll(codeText, "\n", " ")
				sb.WriteString(strings.Join(strings.Fields(normalizedCode), " "))
				// Add a space after the entire code block.
				sb.WriteString(" ")
				// We've handled this node and its children, so we skip to the next sibling.
				return
			}

			// --- Default Traversal ---

			// Recursively traverse children for all other elements.
			traverse(node.Contents())

			// After processing an element and its children, add a space if it's a separator tag.
			if separatorTags[goquery.NodeName(node)] {
				sb.WriteString(" ")
			}
		})
	}

	traverse(doc.Selection)

	// Finally, clean up any resulting multiple spaces and trim the string.
	return strings.Join(strings.Fields(sb.String()), " "), nil
}
