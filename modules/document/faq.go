/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"net/http"
	"strings"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common/fingerprint"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// FAQ knowledge entries (W5): an FAQ entry is a plain document
// (type=faq) whose INDEX content is compiled from the standard question
// and its similar phrasings — the answer and the negative questions are
// metadata, never indexed text. Retrieval: BM25 over the compiled
// questions, negative questions knock their entry out entirely (a user
// asking exactly what the entry explicitly is NOT about should not get
// it), and a normalized exact hit on any phrasing answers directly
// (exact=true, the standard answer attached — the model may adopt it).
//
// Idempotency rides the existing fingerprint: the compiled question text
// IS the content, so the same question set hashes identically and the
// create path returns the existing row instead of stacking twins.

// faqDocType is the document type marker for FAQ entries.
const faqDocType = "faq"

// FAQEntry is the structured FAQ payload, stored under metadata["faq"].
type FAQEntry struct {
	Standard string   `json:"standard"`
	Similar  []string `json:"similar,omitempty"`
	Negative []string `json:"negative,omitempty"`
	Answer   string   `json:"answer,omitempty"`
}

// compileFAQContent builds the indexable text: every question phrasing
// that SHOULD retrieve this entry. Negative questions and the answer stay
// out — negatives must not retrieve, the answer is returned (not searched).
func compileFAQContent(entry *FAQEntry) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(entry.Standard))
	for _, s := range entry.Similar {
		if q := strings.TrimSpace(s); q != "" {
			b.WriteString("\n")
			b.WriteString(q)
		}
	}
	return b.String()
}

// normalizeFAQQuestion reuses the D1 normalization chain (NFKC width/case
// folding) and goes one step further for question keys: ALL whitespace is
// stripped — question phrasings routinely carry stray spaces in CJK, and
// an exact-match key must not miss because of them.
func normalizeFAQQuestion(q string) string {
	n := fingerprint.Normalize(q)
	var b strings.Builder
	for _, r := range n {
		switch r {
		case ' ', '\t', '\n', '\r', ' ', '　':
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// faqQuestionSet returns every phrasing (standard + similar) normalized,
// for exact-match checks.
func faqQuestionSet(entry *FAQEntry) map[string]string {
	// normalized phrasing → display phrasing (standard preferred)
	out := map[string]string{}
	for _, s := range entry.Similar {
		if n := normalizeFAQQuestion(s); n != "" {
			out[n] = s
		}
	}
	if n := normalizeFAQQuestion(entry.Standard); n != "" {
		out[n] = entry.Standard
	}
	return out
}

// faqCreateHandler creates FAQ entries: POST /document/faq with either one
// entry or a list, plus the datasource scope. Same compiled content in the
// same datasource returns the existing row (fingerprint idempotency).
func (h *APIHandler) faqCreateHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body := util.MapStr{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	datasourceID, _ := body["datasource_id"].(string)

	rawEntries, _ := body["entries"].([]interface{})
	if len(rawEntries) == 0 {
		// single-entry convenience shape
		if e := parseFAQEntry(body); e != nil {
			rawEntries = append(rawEntries, util.MapStr{"standard": e.Standard})
			body["entries"] = rawEntries
		}
	}
	if len(rawEntries) == 0 {
		h.WriteError(w, "entries (or standard) required", http.StatusBadRequest)
		return
	}

	created := make([]util.MapStr, 0, len(rawEntries))
	skipped := 0
	for _, raw := range rawEntries {
		m, ok := raw.(map[string]interface{})
		if !ok {
			skipped++
			continue
		}
		entry := parseFAQEntry(m)
		if entry == nil || strings.TrimSpace(entry.Standard) == "" {
			skipped++
			continue
		}
		doc := faqDocument(entry, datasourceID, req)
		if faqExistsSameContent(doc) {
			skipped++
			created = append(created, util.MapStr{"result": "duplicate", "standard": entry.Standard})
			continue
		}
		ctx := orm.NewContextWithParent(req.Context())
		ctx.Refresh = orm.WaitForRefresh
		if err := orm.Create(ctx, doc); err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		created = append(created, util.MapStr{"result": "created", "id": doc.ID, "standard": entry.Standard})
	}

	h.WriteJSON(w, util.MapStr{
		"result":     "ok",
		"created":    created,
		"duplicates": skipped,
	}, 200)
}

func parseFAQEntry(m map[string]interface{}) *FAQEntry {
	e := &FAQEntry{}
	e.Standard, _ = m["standard"].(string)
	e.Similar = stringSliceOf(m["similar"])
	e.Negative = stringSliceOf(m["negative"])
	e.Answer, _ = m["answer"].(string)
	return e
}

// faqDocument compiles an entry into its persisted document: the content
// is the question text only; the structured entry (answers, negatives)
// rides in metadata.
func stringSliceOf(v interface{}) []string {
	items, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func faqDocument(entry *FAQEntry, datasourceID string, req *http.Request) *core.Document {
	content := compileFAQContent(entry)
	doc := &core.Document{
		Title:    strings.TrimSpace(entry.Standard),
		Content:  content,
		Type:     faqDocType,
		Category: "faq",
		Status:   core.DocumentStatusCompleted,
	}
	doc.Source = core.DataSourceReference{ID: datasourceID, Name: "FAQ", Type: faqDocType}
	doc.Metadata = util.MapStr{"faq": entry}
	// the fingerprint hook stamps content_hash at create; pre-stamp here so
	// the duplicate check below and the stored row agree bit for bit
	if h, s, ok := fingerprint.Compute(content); ok {
		doc.ContentHash, doc.ContentSimhash = h, int64(s)
	}
	return doc
}

// faqExistsSameContent checks the dedup gate at write time (the same
// question set in the same datasource must not stack twins).
func faqExistsSameContent(doc *core.Document) bool {
	if doc.ContentHash == "" {
		return false
	}
	octx := orm.NewContext()
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.Document{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(1).
		Filter(
			orm.TermQuery("content_hash", doc.ContentHash),
			orm.TermQuery("type", faqDocType),
			orm.TermQuery("source.id", doc.Source.ID),
		))
	if err != nil {
		return false
	}
	hits, _, _ := elastic.DecodeHits[core.Document](res)
	return len(hits) > 0
}

// faqSearchHandler answers FAQ lookups: GET /query/_faq?q=...&datasource=...
// A normalized exact hit on any phrasing returns the standard answer with
// exact=true; otherwise the ranked BM25 hits come back minus entries whose
// negative questions match the query exactly.
func (h *APIHandler) faqSearchHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	query := strings.TrimSpace(req.URL.Query().Get("q"))
	if query == "" {
		h.WriteError(w, "q is required", http.StatusBadRequest)
		return
	}
	datasource := req.URL.Query().Get("datasource")
	size := h.GetIntOrDefault(req, "size", 10)
	if size > maxSearchPageSize {
		size = maxSearchPageSize
	}

	octx := orm.NewContextWithParent(req.Context())
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.Document{})

	builder := orm.NewQuery().From(0).Size(size)
	builder.Query(query)
	builder.DefaultQueryField("title^20", "title.pinyin^8", "combined_fulltext")
	builder.Filter(orm.TermQuery("type", faqDocType))
	if datasource != "" {
		builder.Filter(orm.TermQuery("source.id", datasource))
	}

	out := &elastic.SearchResponseWithMeta[core.Document]{}
	res, err := orm.SearchV2(octx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if raw, ok := res.Payload.([]byte); ok && len(raw) > 0 {
		util.MustFromJSONBytes(raw, out)
	}

	normalized := normalizeFAQQuestion(query)
	hits := make([]util.MapStr, 0, len(out.Hits.Hits))
	for i := range out.Hits.Hits {
		doc := &out.Hits.Hits[i].Source
		entry := faqEntryOf(doc)
		if entry == nil {
			continue
		}
		// negative gate: the user asked exactly what this entry says it is
		// NOT about — the whole entry is out, not just demoted
		if faqNegativeHit(entry, normalized) {
			continue
		}
		hit := util.MapStr{
			"id":    out.Hits.Hits[i].ID,
			"title": doc.Title,
			"score": out.Hits.Hits[i].Score,
		}
		if entry.Answer != "" {
			hit["answer"] = entry.Answer
		}
		hits = append(hits, hit)

		// exact: the query normalized-equals any phrasing — the standard
		// answer may be adopted directly
		if _, ok := faqQuestionSet(entry)[normalized]; ok {
			h.WriteJSON(w, util.MapStr{
				"result": "ok",
				"exact":  true,
				"entry":  hit,
				"total":  1,
			}, 200)
			return
		}
	}

	h.WriteJSON(w, util.MapStr{
		"result": "ok",
		"exact":  false,
		"hits":   hits,
		"total":  len(hits),
	}, 200)
}

// faqEntryOf extracts the structured entry from a stored document.
func faqEntryOf(doc *core.Document) *FAQEntry {
	raw, ok := doc.Metadata["faq"]
	if !ok {
		return nil
	}
	b, err := util.ToJSONBytes(raw)
	if err != nil {
		return nil
	}
	entry := &FAQEntry{}
	if err := util.FromJSONBytes(b, entry); err != nil {
		return nil
	}
	return entry
}

// faqNegativeHit reports whether the normalized query exactly equals one
// of the entry's negative questions.
func faqNegativeHit(entry *FAQEntry, normalized string) bool {
	for _, n := range entry.Negative {
		if normalizeFAQQuestion(n) == normalized && normalized != "" {
			return true
		}
	}
	return false
}
