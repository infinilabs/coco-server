/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	llmmodule "infini.sh/coco/modules/llm"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"github.com/tmc/langchaingo/llms"
)

// Document amendment (W11): an uploaded file acts as EVIDENCE against an
// existing document — the model aligns the attachment's claims with the
// current content and produces a structured diff (corrections,
// supplements, obsolescence), filed as a governance proposal. The human
// applies what survives review through the ordinary document edit —
// which (W0.1) re-chunks, re-embeds and re-summarizes automatically. The
// model never writes the document.

const (
	amendMaxUploadBytes = 20 << 20 // 20MB evidence cap
	amendMaxDocChars    = 12000    // current-content excerpt cap for the prompt
	amendMaxAttachChars = 12000    // attachment excerpt cap for the prompt
	amendMaxItems       = 8
	amendProposalType   = "document_amendment"
)

// amendDiffItem is one aligned difference.
type amendDiffItem struct {
	Action  string `json:"action"`  // correct | supplement | obsolete
	Claim   string `json:"claim"`   // what the attachment says
	Current string `json:"current"` // what the document says now ("" for supplement)
	Suggest string `json:"suggest"` // proposed replacement/insertion
	Excerpt string `json:"excerpt"` // attachment evidence excerpt
}

const amendSystemPrompt = `You align an uploaded EVIDENCE document against an existing knowledge document.
Compare the evidence with the current content and return ONLY a JSON array of diff items, each:
{"action": "correct" | "supplement" | "obsolete",
 "claim": "<what the evidence states>",
 "current": "<the contradicting/outdated passage in the current document, empty for supplement>",
 "suggest": "<the corrected or new text to incorporate>",
 "excerpt": "<short verbatim quote from the evidence supporting the claim>"}
Rules: "correct" = evidence contradicts current content; "supplement" = evidence adds new information;
"obsolete" = evidence shows current content is outdated/wrong. Max %d items, strongest first.
No explanations, just the JSON array (possibly empty).`

var amendFenceRe = regexp.MustCompile("(?s)```[a-zA-Z]*\\n?(.*?)```")

// amendHandler accepts multipart "file", extracts its text (Tika when
// configured, raw text otherwise), runs the alignment, files the
// proposal: POST /document/:doc_id/_amend
func (h *APIHandler) amendHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	docID := ps.ByName("doc_id")

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "document")
	doc := core.Document{}
	doc.ID = docID
	exists, err := orm.GetV2(ctx, &doc)
	if err != nil || !exists {
		h.WriteOpRecordNotFoundJSON(w, docID)
		return
	}

	file, header, err := req.FormFile("file")
	if err != nil {
		h.WriteError(w, "multipart 'file' field required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	if header.Size > amendMaxUploadBytes {
		h.WriteError(w, "evidence file too large (20MB cap)", http.StatusBadRequest)
		return
	}

	tmp, err := os.CreateTemp("", "coco-amend-*"+filepath.Ext(header.Filename))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, file); err != nil {
		tmp.Close()
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmp.Close()

	attachmentText, extractErr := amendExtractText(req.Context(), tmp.Name())
	if extractErr != nil || strings.TrimSpace(attachmentText) == "" {
		note := "attachment text extraction failed"
		if extractErr != nil {
			note += ": " + extractErr.Error()
		}
		h.WriteError(w, note, http.StatusUnprocessableEntity)
		return
	}

	items, alignErr := alignAmendment(req.Context(), doc.Content, attachmentText)
	if alignErr != nil {
		log.Warnf("amendment: alignment failed for [%s]: %v", docID, alignErr)
		h.WriteError(w, "alignment model unavailable: "+alignErr.Error(), http.StatusServiceUnavailable)
		return
	}
	if len(items) == 0 {
		h.WriteOKJSON(w, util.MapStr{"result": "no differences found", "items": []amendDiffItem{}})
		return
	}

	// file the proposal into the governance queue — human gate first
	proposal := &core.WikiGovernanceProposal{
		Type:         amendProposalType,
		Status:       core.WikiGovernanceOpen,
		ArticleTitle: doc.Title,
		Reason:       fmt.Sprintf("evidence file %q suggests %d change(s)", header.Filename, len(items)),
	}
	proposal.ID = core.AnchoredProposalID("doc:"+docID, amendProposalType, amendmentGeneration(docID)+1)
	proposal.Evidence = util.MapStr{
		"doc_id":             docID,
		"attachment":         header.Filename,
		"cascade":            "document-amendment",
		"items":              items,
		"attachment_excerpt": truncateRunes(attachmentText, 500),
	}
	pctx := orm.NewContextWithParent(req.Context())
	pctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	pctx.Refresh = orm.WaitForRefresh
	if err := orm.Create(pctx, proposal); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteOKJSON(w, util.MapStr{
		"result":     "proposal filed — review in the governance queue",
		"proposalId": proposal.ID,
		"items":      items,
		"note":       "apply accepted changes through the ordinary document edit; reprocessing follows automatically",
	})
}

// amendmentGeneration counts existing proposals for anchor numbering.
func amendmentGeneration(docID string) int {
	octx := orm.NewContext()
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiGovernanceProposal{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(100).
		Filter(orm.TermQuery("type", amendProposalType)))
	if err != nil || res == nil {
		return 0
	}
	count := 0
	out := &struct {
		Hits struct {
			Hits []struct {
				Source core.WikiGovernanceProposal `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}{}
	if raw, ok := res.Payload.([]byte); ok {
		if json.Unmarshal(raw, out) == nil {
			for _, hh := range out.Hits.Hits {
				ev, _ := hh.Source.Evidence["doc_id"].(string)
				if ev == docID {
					count++
				}
			}
		}
	}
	return count
}

// amendExtractText pulls text out of the evidence file: Tika when
// configured (the pipeline's endpoint), raw bytes as the text fallback.
func amendExtractText(ctx context.Context, path string) (string, error) {
	endpoint := common.AppConfig().DocumentProcessing.EffectiveTikaEndpoint()
	if text, err := tikaTextPlain(ctx, endpoint, path); err == nil && strings.TrimSpace(text) != "" {
		return text, nil
	} else if err != nil {
		log.Debugf("amendment: tika extraction failed (%v), raw fallback", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// tikaTextPlain posts the file to Tika for text/plain extraction — a
// local re-implementation of fileproc's call, because importing the
// processors package from here closes an import cycle (fileproc pulls
// the connector stack).
func tikaTextPlain(ctx context.Context, endpoint, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, strings.TrimSuffix(endpoint, "/")+"/tika", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tika returned %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return string(b), err
}

// alignAmendment runs the evidence-vs-document alignment.
func alignAmendment(ctx context.Context, docContent, attachmentText string) ([]amendDiffItem, error) {
	m := llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	if m == nil {
		return nil, fmt.Errorf("no default language model configured")
	}
	llm, err := langchain.SimplyGetLLM(m.ProviderID, m.ID, "")
	if err != nil {
		return nil, err
	}
	messages := []llms.MessageContent{
		langchain.SystemTextParts(fmt.Sprintf(amendSystemPrompt, amendMaxItems)),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf(
			"CURRENT DOCUMENT:\n%s\n\nEVIDENCE:\n%s",
			truncateRunes(docContent, amendMaxDocChars),
			truncateRunes(attachmentText, amendMaxAttachChars))),
	}
	resp, err := llm.GenerateContent(ctx, messages, llms.WithMaxTokens(1500))
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty alignment response")
	}
	return parseAmendOutput(resp.Choices[0].Content)
}

func parseAmendOutput(s string) ([]amendDiffItem, error) {
	if fenced := amendFenceRe.FindStringSubmatch(s); len(fenced) > 1 {
		s = fenced[1]
	}
	start, end := strings.Index(s, "["), strings.LastIndex(s, "]")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON array in alignment output")
	}
	var items []amendDiffItem
	if err := json.Unmarshal([]byte(s[start:end+1]), &items); err != nil {
		return nil, err
	}
	if len(items) > amendMaxItems {
		items = items[:amendMaxItems]
	}
	valid := items[:0]
	for _, it := range items {
		switch it.Action {
		case "correct", "supplement", "obsolete":
			if strings.TrimSpace(it.Claim) != "" {
				valid = append(valid, it)
			}
		}
	}
	return valid, nil
}
