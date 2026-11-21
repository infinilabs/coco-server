/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"fmt"
	"regexp"
	"strings"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
)

// Figures on compiled pages (图文并茂): embedded-image attachments of the
// cited source documents join the page inline — but only when their OCR
// text overlaps the section they would illustrate. The same OCR text is
// supplementary material everywhere else: StripImageMarkers keeps it out
// of the compilation corpus so scan noise never spawns standalone pages
// or entities.

// imageMarkerRe matches the [[Image(uuid\tOCR text)]] markers the document
// pipeline (text_attachment_extraction) inlines for embedded figures.
var imageMarkerRe = regexp.MustCompile(`\[\[Image\([^\]]*\)\]\]`)

// StripImageMarkers removes embedded-figure markers from document content.
// The OCR text inside a marker is attachment-derived supplementary text —
// compiled pages and entities must be born from real document prose, not
// from watermark/table-noise OCR.
func StripImageMarkers(content string) string {
	if !strings.Contains(content, "[[Image(") {
		return content
	}
	return strings.TrimSpace(imageMarkerRe.ReplaceAllString(content, " "))
}

const (
	pageImageMaxPerPage     = 3
	pageImageMaxDocs        = 30
	pageImageMaxAttachments = 60
	// pageImageMinOverlap: distinct shared word-bigrams (or one ≥4-letter
	// latin token) between a figure's OCR text and its section heading +
	// page title before the figure is considered relevant
	pageImageMinOverlap  = 2
	pageImageAltMaxRunes = 60
)

var imageAttachmentExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".bmp": true, ".svg": true,
}

// topicImage is one relevance-gated figure placed after its section body.
type topicImage struct {
	SectionID string
	URL       string
	Alt       string
	DocID     string
	DocTitle  string
}

func isImageAttachment(a core.Attachment) bool {
	if a.Deleted || strings.TrimSpace(a.URL) == "" {
		return false
	}
	if strings.HasPrefix(a.MimeType, "image/") {
		return true
	}
	name := strings.ToLower(a.Name)
	for ext := range imageAttachmentExt {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// wordBigrams collects rune-bigrams per whitespace-separated run, never
// across run boundaries (cross-space bigrams match noise like "e T").
func wordBigrams(s string) map[string]bool {
	out := map[string]bool{}
	for _, field := range strings.Fields(strings.ToLower(s)) {
		r := []rune(field)
		for i := 0; i+1 < len(r); i++ {
			out[string(r[i:i+2])] = true
		}
	}
	return out
}

// figureOverlaps reports whether a figure's OCR text is relevant to the
// section it would illustrate: at least pageImageMinOverlap distinct shared
// word-bigrams, or one shared latin token of 4+ letters.
func figureOverlaps(ocr, heading, title string) bool {
	ocr = strings.ToLower(ocr)
	if strings.TrimSpace(ocr) == "" {
		return false
	}
	target := wordBigrams(heading + " " + title)
	if len(target) == 0 {
		return false
	}
	hits := 0
	for bg := range wordBigrams(ocr) {
		if target[bg] {
			hits++
			if hits >= pageImageMinOverlap {
				return true
			}
		}
	}
	for _, f := range strings.Fields(ocr) {
		f = strings.Trim(f, ",.;:!?()\"'[]")
		if len(f) >= 4 && target[f] {
			return true
		}
	}
	return false
}

// imageAlt condenses OCR text into a markdown-safe alt/caption.
func imageAlt(ocr, fallback string) string {
	s := strings.Join(strings.Fields(ocr), " ")
	s = strings.Map(func(r rune) rune {
		if r == '[' || r == ']' || r == '\n' || r == '\t' {
			return ' '
		}
		return r
	}, s)
	if r := []rune(s); len(r) > pageImageAltMaxRunes {
		s = string(r[:pageImageAltMaxRunes]) + "…"
	}
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

type pageFigure struct {
	url, ocr, docID string
}

// collectSectionImages picks at most one relevant figure per section (and
// at most pageImageMaxPerPage per page) from the image attachments of the
// documents each section cites. Sections without citations, figures
// without OCR text, or figures whose OCR does not overlap the section
// heading are all skipped — relevance is proven, not assumed.
func collectSectionImages(ctx *orm.Context, title string, sections []core.WikiSection, docTitles map[string]string) []topicImage {
	// cited docs in first-appearance order, capped
	docOrder := make([]string, 0, 16)
	seenDoc := map[string]bool{}
	for _, s := range sections {
		for _, d := range s.DocIDs {
			if d != "" && !seenDoc[d] {
				seenDoc[d] = true
				docOrder = append(docOrder, d)
			}
		}
	}
	if len(docOrder) == 0 || ctx == nil {
		return nil
	}
	if len(docOrder) > pageImageMaxDocs {
		docOrder = docOrder[:pageImageMaxDocs]
	}

	// documents → their attachment id lists; the inversion below keys each
	// attachment back to its owning doc, which is what ties a figure to the
	// section citing that doc
	dctx := orm.NewContextWithParent(ctx)
	dctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(dctx, &core.Document{})
	res, err := orm.SearchV2(dctx, orm.NewQuery().Size(len(docOrder)).
		Filter(orm.TermsQuery("id", docOrder)).
		Include("id", "attachments"))
	if err != nil || res == nil {
		return nil
	}
	docs, _, derr := elastic.DecodeHits[core.Document](res)
	if derr != nil {
		return nil
	}
	attIDs := make([]string, 0, pageImageMaxAttachments)
	figuresByDoc := map[string][]pageFigure{}
	attOwner := map[string]string{}
	for _, d := range docs {
		if len(d.Attachments) == 0 {
			continue
		}
		for _, a := range d.Attachments {
			if len(attIDs) >= pageImageMaxAttachments {
				break
			}
			if _, dup := attOwner[a]; !dup {
				attOwner[a] = d.ID
				attIDs = append(attIDs, a)
			}
		}
	}
	if len(attIDs) == 0 {
		return nil
	}

	actx := orm.NewContextWithParent(ctx)
	actx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(actx, &core.Attachment{})
	ares, aerr := orm.SearchV2(actx, orm.NewQuery().Size(len(attIDs)).
		Filter(orm.TermsQuery("id", attIDs)))
	if aerr != nil || ares == nil {
		return nil
	}
	atts, _, aerr := elastic.DecodeHits[core.Attachment](ares)
	if aerr != nil {
		return nil
	}
	for _, a := range atts {
		if !isImageAttachment(a) {
			continue
		}
		docID := attOwner[a.ID]
		if docID == "" {
			continue
		}
		figuresByDoc[docID] = append(figuresByDoc[docID], pageFigure{url: a.URL, ocr: a.Text, docID: docID})
	}
	if len(figuresByDoc) == 0 {
		return nil
	}

	images := make([]topicImage, 0, pageImageMaxPerPage)
	usedFigure := map[string]bool{}
	for _, s := range sections {
		if len(images) >= pageImageMaxPerPage {
			break
		}
		for _, d := range s.DocIDs {
			picked := false
			for _, f := range figuresByDoc[d] {
				if usedFigure[f.url] || !figureOverlaps(f.ocr, s.Heading, title) {
					continue
				}
				usedFigure[f.url] = true
				images = append(images, topicImage{
					SectionID: s.ID,
					URL:       f.url,
					Alt:       imageAlt(f.ocr, docTitles[d]),
					DocID:     d,
					DocTitle:  docTitles[d],
				})
				picked = true
				break
			}
			if picked {
				break
			}
		}
	}
	return images
}

// renderSectionImage emits the markdown for one placed figure: the image
// with its OCR caption plus a pointer at the source document.
func renderSectionImage(img topicImage) string {
	alt := strings.TrimSpace(img.Alt)
	if alt == "" {
		alt = img.DocTitle
	}
	source := ""
	if img.DocID != "" {
		label := img.DocTitle
		if label == "" {
			label = img.DocID
		}
		source = fmt.Sprintf(" — [%s](#/preview/document/%s)", label, img.DocID)
	}
	return fmt.Sprintf("![%s](%s)\n\n*图：%s%s*", alt, img.URL, alt, source)
}
