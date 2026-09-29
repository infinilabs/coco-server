/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package extract_entities

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	llmmodule "infini.sh/coco/modules/llm"
	"infini.sh/coco/modules/wiki"
	utils "infini.sh/coco/plugins/processors"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const ProcessorName = "extract_entities"

const MinimumModelContextLength = 4000

// DefaultEntityTypes is the fallback ontology vocabulary when the pipeline
// config doesn't pin one (design doc §3.5: the real vocabulary lives in
// WikiOntologySchema; the processor takes a flat list for the POC).
var DefaultEntityTypes = []string{
	"person", "organization", "product", "contract",
	"campaign", "store", "supplier", "concept", "event", "location",
}

type Config struct {
	MessageField param.ParaKey      `config:"message_field"`
	OutputQueue  *queue.QueueConfig `config:"output_queue"`

	ModelProviderID    string `config:"model_provider"`
	ModelName          string `config:"model"`
	ModelContextLength uint32 `config:"model_context_length"`

	// ontology vocabulary the LLM may choose types from
	EntityTypes []string `config:"entity_types"`
	// hard cap on extracted entities per document
	MaxEntities int `config:"max_entities"`
	// when set, a draft entity page is created in this KB for every
	// newly-proposed entity (design doc D1)
	WikiKbID string `config:"wiki_kb_id"`

	// Language for LLM-generated content (BCP 47 tag, e.g. "zh-CN")
	LLMGenerationLang string `config:"llm_generation_lang"`
}

type ExtractEntitiesProcessor struct {
	config             *Config
	outputQueue        *queue.QueueConfig
	removeThinkPattern *regexp.Regexp
}

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		MessageField: core.PipelineContextDocuments,
		MaxEntities:  15,
	}
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of %s processor: %s", ProcessorName, err)
	}

	if cfg.LLMGenerationLang == "" {
		if appCfg := common.AppConfig(); appCfg.DocumentProcessing != nil && appCfg.DocumentProcessing.LLMGenerationLanguage != "" {
			cfg.LLMGenerationLang = appCfg.DocumentProcessing.LLMGenerationLanguage
		}
	}
	cfg.LLMGenerationLang = utils.ValidateAndNormalizeLLMLang(ProcessorName, cfg.LLMGenerationLang)

	if cfg.MessageField == "" {
		cfg.MessageField = core.PipelineContextDocuments
	}
	if len(cfg.EntityTypes) == 0 {
		cfg.EntityTypes = DefaultEntityTypes
	}
	if cfg.MaxEntities <= 0 {
		cfg.MaxEntities = 15
	}

	if cfg.ModelContextLength > 0 && cfg.ModelContextLength < MinimumModelContextLength {
		panic("Model's context length is too low")
	}

	processor := ExtractEntitiesProcessor{config: &cfg}
	if cfg.OutputQueue != nil {
		processor.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}
	processor.removeThinkPattern = regexp.MustCompile(`(?s)`)
	return &processor, nil
}

func (processor *ExtractEntitiesProcessor) Name() string {
	return ProcessorName
}

// extractedEntity mirrors the LLM JSON contract; relations reference target
// entities by NAME and are resolved to ids after disambiguation.
type extractedEntity struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Aliases    []string               `json:"aliases"`
	Properties map[string]interface{} `json:"properties"`
	Relations  []struct {
		Relation string `json:"relation"`
		Target   string `json:"target"`
	} `json:"relations"`
	Evidence string `json:"evidence"`
}

type extractionResult struct {
	Entities []extractedEntity `json:"entities"`
}

func (processor *ExtractEntitiesProcessor) Process(ctx *pipeline.Context) error {
	obj := ctx.Get(processor.config.MessageField)
	if obj == nil {
		log.Warnf("processor [%s] receives an empty pipeline context", processor.Name())
		return nil
	}
	messages, ok := obj.([]queue.Message)
	if !ok {
		log.Warnf("processor [%s] context value is not []queue.Message", processor.Name())
		return nil
	}
	if len(messages) == 0 {
		return nil
	}

	modelId := llmmodule.ResolveModel(core.LLMTypeLanguage, &core.ModelId{ProviderID: processor.config.ModelProviderID, ID: processor.config.ModelName})
	if modelId == nil {
		return fmt.Errorf("[%s] no language model configured: set model_provider/model in pipeline config or configure a default language model in settings", processor.Name())
	}
	provider, err := common.GetModelProvider(modelId.ProviderID)
	if err != nil {
		return err
	}
	llm := langchain.GetLLM(provider.BaseURL, provider.APIType, modelId.ID, provider.APIKey, "")
	llmCtx, cancelFunc := context.WithCancel(ctx.Context)
	defer cancelFunc()

	// ontology vocabulary (phase O2): schema-constrained extraction — kb
	// override first, tenant default as fallback; nil means free-form POC list
	schema := wiki.ResolveOntologySchema(ctx.Context, processor.config.WikiKbID)

	enqueued := make(map[int]bool)

	for i := range messages {
		if global.ShuttingDown() {
			return errors.New("shutting down")
		}
		message := &messages[i]
		doc := core.Document{}
		if err := util.FromJSONBytes(message.Data, &doc); err != nil {
			log.Error("error on handle document:", i, err)
			continue
		}

		material := buildExtractionMaterial(&doc)
		if material == "" {
			log.Debugf("[%s] document [%s/%s] has no extractable text, skipping", processor.Name(), doc.Title, doc.ID)
		} else {
			log.Infof("processor [%s] start extracting entities for document [%s/%s]", processor.Name(), doc.Title, doc.ID)
			start := time.Now()
			entityIDs, err := processor.extractAndLink(llmCtx, llm, &doc, material, schema)
			if err != nil {
				log.Errorf("[%s] failed to extract entities for document [%s/%s], error [%s]", processor.Name(), doc.Title, doc.ID, err)
			} else if len(entityIDs) > 0 {
				doc.EntityIDs = entityIDs
				log.Infof("[%s] finished extracting entities for doc, %v, %v, elapsed: %v, entities: %v",
					processor.Name(), doc.Title, doc.ID, util.Since(start), entityIDs)
				message.Data = util.MustToJSONBytes(doc)
			}
		}

		if processor.outputQueue != nil {
			if err := queue.Push(processor.outputQueue, message.Data); err != nil {
				log.Errorf("processor [%s] failed to push document [%s/%s] to output queue: %v", processor.Name(), doc.Title, doc.ID, err)
			} else {
				enqueued[i] = true
			}
		}
	}

	if processor.outputQueue != nil {
		for i := range messages {
			if !enqueued[i] {
				if err := queue.Push(processor.outputQueue, messages[i].Data); err != nil {
					log.Errorf("processor [%s] failed to push skipped document [%d] to output queue: %v", processor.Name(), i, err)
				}
			}
		}
	}
	return nil
}

// buildExtractionMaterial picks the richest text a document carries:
// AI insights first (summary/insights pipeline products), then the raw body.
func buildExtractionMaterial(doc *core.Document) string {
	parts := []string{}
	if doc.Title != "" {
		parts = append(parts, "Title: "+doc.Title)
	}
	if doc.Summary != "" {
		parts = append(parts, "Summary: "+doc.Summary)
	}
	if doc.AiInsights.Text != "" {
		parts = append(parts, "Analysis: "+doc.AiInsights.Text)
	}
	return strings.Join(parts, "\n\n")
}

// extractAndLink runs the LLM extraction, disambiguates against existing
// entities, proposes the unknown ones and returns the linked entity ids
// (also back-linking the document onto each entity's sources).
func (processor *ExtractEntitiesProcessor) extractAndLink(ctx context.Context, llm llms.Model, doc *core.Document, material string, schema *wiki.OntologySchemaDoc) ([]string, error) {
	result, err := extractEntitiesFromText(ctx, llm, material, processor.config, schema, processor.removeThinkPattern)
	if err != nil {
		return nil, err
	}
	if len(result.Entities) == 0 {
		return nil, nil
	}
	if len(result.Entities) > processor.config.MaxEntities {
		result.Entities = result.Entities[:processor.config.MaxEntities]
	}

	ormCtx := orm.NewContext()
	ormCtx.Set(orm.DirectWriteWithoutPermissionCheck, true) // pipeline context has no session user

	resolved := make([]*core.WikiEntity, 0, len(result.Entities))
	seen := map[string]bool{}
	for i := range result.Entities {
		e := &result.Entities[i]
		e.Name = strings.TrimSpace(e.Name)
		if e.Name == "" {
			continue
		}
		entity, created, err := resolveOrCreateEntity(ormCtx, e, doc, processor.config.EntityTypes, schema, processor.config.WikiKbID)
		if err != nil {
			log.Warnf("[%s] failed to resolve entity %q: %v", ProcessorName, e.Name, err)
			continue
		}
		if seen[entity.ID] {
			continue
		}
		seen[entity.ID] = true
		resolved = append(resolved, entity)
		if created {
			log.Debugf("[%s] proposed new entity [%s/%s] from document %s", ProcessorName, entity.Type, entity.Name, doc.ID)
			if processor.config.WikiKbID != "" {
				if err := createDraftEntityPage(ormCtx, entity, processor.config.WikiKbID); err != nil {
					log.Warnf("[%s] failed to create draft page for entity %q: %v", ProcessorName, entity.Name, err)
				}
			}
		}
	}

	// second pass: relation targets are names — resolve them against the
	// batch we just resolved, then persist typed edges
	if err := writeRelationEdges(ormCtx, resolved, result.Entities, doc.ID); err != nil {
		log.Warnf("[%s] failed to write relation edges for document %s: %v", ProcessorName, doc.ID, err)
	}

	ids := make([]string, 0, len(resolved))
	for _, e := range resolved {
		ids = append(ids, e.ID)
	}
	return ids, nil
}

// resolveOrCreateEntity disambiguates by exact name, then alias match
// (case-insensitive retry); unmatched extractions become proposed entities
// with the document recorded as provenance (design doc B2).
func resolveOrCreateEntity(ctx *orm.Context, extracted *extractedEntity, doc *core.Document, allowedTypes []string, schema *wiki.OntologySchemaDoc, kbID string) (*core.WikiEntity, bool, error) {
	orm.WithModel(ctx, &core.WikiEntity{})

	extractedType := strings.ToLower(strings.TrimSpace(extracted.Type))
	if existing := findEntityByName(ctx, extracted.Name); existing != nil {
		// same name, conflicting types: keep merging the provenance but let
		// the governance queue ask the human whether to split (O2)
		if existing.Type != "" && extractedType != "" && existing.Type != extractedType {
			wiki.FileEntityGovernanceProposal(kbID, existing, core.WikiGovernanceEntityConflict,
				fmt.Sprintf("document %q mentions %q as %q, existing entity is %q", doc.Title, extracted.Name, extractedType, existing.Type),
				util.MapStr{"doc_id": doc.ID, "existing_type": existing.Type, "extracted_type": extractedType})
		}
		appendEntitySource(existing, extracted, doc)
		if err := orm.Update(ctx, existing); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}
	for _, alias := range extracted.Aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		if existing := findEntityByName(ctx, alias); existing != nil {
			if existing.Type != "" && extractedType != "" && existing.Type != extractedType {
				wiki.FileEntityGovernanceProposal(kbID, existing, core.WikiGovernanceEntityConflict,
					fmt.Sprintf("document %q mentions alias %q of %q as %q, existing entity is %q", doc.Title, alias, extracted.Name, extractedType, existing.Type),
					util.MapStr{"doc_id": doc.ID, "alias": alias, "existing_type": existing.Type, "extracted_type": extractedType})
			}
			appendEntitySource(existing, extracted, doc)
			if err := orm.Update(ctx, existing); err != nil {
				return nil, false, err
			}
			return existing, false, nil
		}
	}

	if schema != nil && len(schema.EntityTypes) > 0 {
		allowedTypes = make([]string, 0, len(schema.EntityTypes))
		for i := range schema.EntityTypes {
			allowedTypes = append(allowedTypes, schema.EntityTypes[i].Name)
		}
	}
	entityType := extractedType
	if !util.AnyInArrayEquals(lowerAll(allowedTypes), entityType) || entityType == "" {
		entityType = "concept" // unknown vocabulary: degrade instead of dropping
	}

	entity := &core.WikiEntity{
		Type:       entityType,
		Name:       extracted.Name,
		Aliases:    normalizeAliases(extracted.Aliases),
		Properties: extracted.Properties,
		Status:     core.WikiEntityProposed, // human review required (D1)
		Sources:    []core.WikiSourceReference{documentSourceRef(doc, extracted.Evidence)},
	}
	if schema != nil {
		if dropped := wiki.SanitizeEntityAgainstSchema(entity, schema); len(dropped) > 0 {
			log.Debugf("[%s] schema sanitize dropped from %q: %v", ProcessorName, extracted.Name, dropped)
		}
	}
	if err := orm.Create(ctx, entity); err != nil {
		return nil, false, err
	}
	return entity, true, nil
}

func lowerAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = strings.ToLower(s)
	}
	return out
}

// createDraftEntityPage gives a newly-proposed entity its curated page
// stub (design doc D1): draft status, KM agent fills the content later.
func createDraftEntityPage(ctx *orm.Context, entity *core.WikiEntity, kbID string) error {
	builder := orm.NewQuery().
		Size(1).
		Filter(orm.TermQuery("entity_id", entity.ID))
	res, err := orm.SearchV2(ctx, builder)
	if err == nil {
		articles, _, derr := decodeArticleHits(res)
		if derr == nil && len(articles) > 0 {
			return nil // page already exists
		}
	}
	orm.WithModel(ctx, &core.WikiArticle{})
	article := &core.WikiArticle{
		KbID:        kbID,
		Title:       entity.Name,
		PageType:    core.WikiPageTypeEntity,
		EntityID:    entity.ID,
		Status:      core.WikiArticleDraft,
		AIGenerated: true, // stub originates from the pipeline; humans curate
	}
	if err := orm.Create(ctx, article); err != nil {
		return err
	}
	entity.ArticleID = article.ID
	return orm.Update(ctx, entity)
}

func caseInsensitiveList(types []string) []string {
	out := make([]string, len(types))
	for i, t := range types {
		out[i] = strings.ToLower(t)
	}
	return out
}

// findEntityByName shares the wiki module's portable resolver (exact name,
// then case-insensitive, then alias with backend-portable fallback).
func findEntityByName(ctx *orm.Context, name string) *core.WikiEntity {
	return wiki.FindEntityByNameOrAlias(ctx, name, "")
}

func decodeArticleHits(res *orm.SearchResult) ([]core.WikiArticle, int64, error) {
	return elastic.DecodeHits[core.WikiArticle](res)
}

// writeRelationEdges resolves each relation's target name to a batch entity
// and upserts the typed edge on the source entity (dedup by
// target+relation+provenance).
func writeRelationEdges(ctx *orm.Context, resolved []*core.WikiEntity, extracted []extractedEntity, docID string) error {
	byName := map[string]*core.WikiEntity{}
	for _, e := range resolved {
		byName[strings.ToLower(e.Name)] = e
		for _, a := range e.Aliases {
			byName[strings.ToLower(a)] = e
		}
	}

	for i := range extracted {
		src := byName[strings.ToLower(strings.TrimSpace(extracted[i].Name))]
		if src == nil {
			continue
		}
		changed := false
		for _, r := range extracted[i].Relations {
			relation := strings.TrimSpace(r.Relation)
			target := byName[strings.ToLower(strings.TrimSpace(r.Target))]
			if relation == "" || target == nil || target.ID == src.ID {
				continue
			}
			if hasRelationEdge(src, target.ID, relation, docID) {
				continue
			}
			src.Relations = append(src.Relations, core.WikiEntityRelation{
				TargetID:   target.ID,
				TargetType: target.Type,
				Relation:   relation,
				Provenance: docID,
			})
			changed = true
		}
		if changed {
			orm.WithModel(ctx, &core.WikiEntity{})
			if err := orm.Update(ctx, src); err != nil {
				return err
			}
		}
	}
	return nil
}

func hasRelationEdge(e *core.WikiEntity, targetID, relation, provenance string) bool {
	for _, r := range e.Relations {
		if r.TargetID == targetID && r.Relation == relation && r.Provenance == provenance {
			return true
		}
	}
	return false
}

// appendEntitySource back-links the document onto a matched entity without
// duplicating provenance on re-sync.
func appendEntitySource(entity *core.WikiEntity, extracted *extractedEntity, doc *core.Document) {
	for _, s := range entity.Sources {
		if s.DocID == doc.ID {
			return
		}
	}
	// merge new aliases discovered in later documents
	for _, alias := range extracted.Aliases {
		alias = strings.TrimSpace(alias)
		if alias != "" && !util.AnyInArrayEquals(entity.Aliases, alias) {
			entity.Aliases = append(entity.Aliases, alias)
		}
	}
	entity.Sources = append(entity.Sources, documentSourceRef(doc, extracted.Evidence))
}

func documentSourceRef(doc *core.Document, evidence string) core.WikiSourceReference {
	ref := core.WikiSourceReference{
		DocID:   doc.ID,
		Title:   doc.Title,
		Excerpt: evidence,
	}
	if doc.Source.ID != "" {
		ref.SourceType = doc.Source.Type
		ref.SourceName = doc.Source.Name
	}
	if doc.URL != "" {
		ref.URL = doc.URL
	}
	return ref
}

func normalizeAliases(aliases []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(aliases))
	for _, a := range aliases {
		a = strings.TrimSpace(a)
		if a == "" || seen[a] {
			continue
		}
		seen[a] = true
		out = append(out, a)
	}
	return out
}

func extractEntitiesFromText(ctx context.Context, llm llms.Model, material string, config *Config, schema *wiki.OntologySchemaDoc, regexpToRemoveThink *regexp.Regexp) (*extractionResult, error) {
	systemPrompt := fmt.Sprintf(
		"You are an expert ontology extractor. Extract entities and typed relations from documents. Your response MUST be in %s.",
		config.LLMGenerationLang)
	userPrompt := buildExtractionPrompt(material, config, schema)

	message := []llms.MessageContent{
		langchain.SystemTextParts(systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	responseBuilder := strings.Builder{}
	_, err := llm.GenerateContent(ctx, message, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		if global.ShuttingDown() {
			ctx.Done()
			return errors.New("shutting down")
		}
		responseBuilder.Write(chunk)
		return nil
	}))
	if err != nil {
		return nil, err
	}

	response := regexpToRemoveThink.ReplaceAllLiteralString(responseBuilder.String(), "")
	return parseEntitiesFromResponse(response)
}

func buildExtractionPrompt(material string, config *Config, schema *wiki.OntologySchemaDoc) string {
	vocabulary, constraints := vocabularySection(config, schema)
	return fmt.Sprintf(
		"Extract entities and relations from the following document.\n\n"+
			"Allowed entity types with their vocabulary:\n%s\n\n"+
			"Requirements:\n"+
			"- Return ONLY a valid JSON object, no markdown fences\n"+
			"- Extract at most %d entities, only clearly mentioned ones\n"+
			"- Use one of the allowed entity types for \"type\"\n"+
			"- \"aliases\": alternate names/s spellings found in the text (may be empty)\n"+
			"- \"properties\": up to 5 short factual attributes as {\"key\": \"value\"}\n"+
			"- \"relations\": links to OTHER entities extracted from the same document as {\"relation\": \"verb_phrase\", \"target\": \"<entity name>\"}\n"+
			"- \"evidence\": one short sentence quoting the mention (in %s)\n"+
			"- Names and aliases MUST be in their original language as they appear\n%s\n"+
			"Format:\n"+
			`{"entities":[{"name":"","type":"","aliases":[],"properties":{},"relations":[{"relation":"","target":""}],"evidence":""}]}`+"\n\n"+
			"Document:\n%s\n\n"+
			"Generate the JSON object now.",
		vocabulary,
		config.MaxEntities,
		config.LLMGenerationLang,
		constraints,
		material,
	)
}

// vocabularySection renders the allowed types for the prompt. With a live
// ontology schema every type carries its declared property keys and relation
// vocabulary (phase O2); without one the flat config list is the POC
// fallback.
func vocabularySection(config *Config, schema *wiki.OntologySchemaDoc) (string, string) {
	if schema == nil || len(schema.EntityTypes) == 0 {
		typesJSON, _ := json.Marshal(config.EntityTypes)
		return string(typesJSON), ""
	}

	var b strings.Builder
	var constraints []string
	for i := range schema.EntityTypes {
		t := &schema.EntityTypes[i]
		fmt.Fprintf(&b, "- %s", t.Name)
		if len(t.Label) > 0 {
			fmt.Fprintf(&b, " (%s)", t.Label)
		}
		if len(t.Properties) > 0 {
			keys := make([]string, 0, len(t.Properties))
			for _, p := range t.Properties {
				if p.Required {
					keys = append(keys, p.Key+"!")
				} else {
					keys = append(keys, p.Key)
				}
			}
			fmt.Fprintf(&b, " | properties: %s", strings.Join(keys, ", "))
		}
		if len(t.Relations) > 0 {
			rels := make([]string, 0, len(t.Relations))
			for _, r := range t.Relations {
				rel := r.Name + "->" + r.TargetType
				if r.Cardinality == "one" {
					rel += " (single)"
				}
				rels = append(rels, rel)
			}
			fmt.Fprintf(&b, " | relations: %s", strings.Join(rels, ", "))
		}
		b.WriteString("\n")
	}
	constraints = append(constraints,
		"- Prefer the listed property keys for \"properties\" when they fit; unknown keys are kept as free-form",
		"- Use ONLY the listed relation names for \"relations\" of each type when possible")
	return b.String(), strings.Join(constraints, "\n")
}

func parseEntitiesFromResponse(response string) (*extractionResult, error) {
	trimmed := strings.TrimSpace(response)
	jsonStart := strings.Index(trimmed, "{")
	jsonEnd := strings.LastIndex(trimmed, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonStart > jsonEnd {
		return nil, fmt.Errorf("no valid JSON object found in response")
	}
	result := extractionResult{}
	if err := json.Unmarshal([]byte(trimmed[jsonStart:jsonEnd+1]), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return &result, nil
}
