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
	"infini.sh/coco/plugins/connectors"
	utils "infini.sh/coco/plugins/processors"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/event"
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
	pipeline.RegisterProcessorPluginWithConfigMetadata(ProcessorName, New, Config{})
	pipeline.RegisterProcessorContract(ProcessorName, utils.DocumentRecordContract(pipeline.GranularityBatchCapable))
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
	// Summary is a one-sentence grounded description of what this entity
	// is, per this document — persisted as WikiEntity.Description.
	Summary string `json:"summary"`
}

type extractionResult struct {
	Entities []extractedEntity `json:"entities"`
}

// extract is the per-document core shared by both conventions.
func (processor *ExtractEntitiesProcessor) extract(llmCtx context.Context, llm llms.Model, schema *wiki.OntologySchemaDoc, doc *core.Document) (bool, error) {
	material := buildExtractionMaterial(doc)
	if material == "" {
		log.Debugf("[%s] document [%s/%s] has no extractable text, skipping", processor.Name(), doc.Title, doc.ID)
		return false, nil
	}
	log.Infof("processor [%s] start extracting entities for document [%s/%s]", processor.Name(), doc.Title, doc.ID)
	start := time.Now()
	entityIDs, err := processor.extractAndLink(llmCtx, llm, doc, material, schema)
	if err != nil {
		log.Errorf("[%s] failed to extract entities for document [%s/%s], error [%s]", processor.Name(), doc.Title, doc.ID, err)
		return false, nil // per-document failure skips the doc, matching legacy
	}
	if len(entityIDs) == 0 {
		return false, nil
	}
	doc.EntityIDs = entityIDs
	log.Infof("[%s] finished extracting entities for doc, %v, %v, elapsed: %v, entities: %v",
		processor.Name(), doc.Title, doc.ID, util.Since(start), entityIDs)
	return true, nil
}

// resolveLLM resolves the language model once per invocation.
func (processor *ExtractEntitiesProcessor) resolveLLM() (llms.Model, error) {
	modelId := llmmodule.ResolveModel(core.LLMTypeLanguage, &core.ModelId{ProviderID: processor.config.ModelProviderID, ID: processor.config.ModelName})
	if modelId == nil {
		return nil, fmt.Errorf("[%s] no language model configured: set model_provider/model in pipeline config or configure a default language model in settings", processor.Name())
	}
	provider, err := common.GetModelProvider(modelId.ProviderID)
	if err != nil {
		return nil, err
	}
	return langchain.GetLLM(provider.BaseURL, provider.APIType, modelId.ID, provider.APIKey, ""), nil
}

// ProcessBatch: record-convention batch channel, lazy LLM resolution.
func (processor *ExtractEntitiesProcessor) ProcessBatch(ctx *pipeline.Context, records []*event.Event) error {
	var llm llms.Model
	resolved := false
	getLLM := func() (llms.Model, error) {
		if resolved {
			return llm, nil
		}
		var err error
		llm, err = processor.resolveLLM()
		if err != nil {
			return nil, err
		}
		resolved = true
		return llm, nil
	}
	llmCtx, cancelFunc := context.WithCancel(ctx.Context)
	defer cancelFunc()
	// ontology schema resolves lazily too — skip-only batches must not
	// require ORM availability
	var schema *wiki.OntologySchemaDoc
	schemaResolved := false
	getSchema := func() *wiki.OntologySchemaDoc {
		if schemaResolved {
			return schema
		}
		schema = wiki.ResolveOntologySchema(ctx.Context, processor.config.WikiKbID)
		schemaResolved = true
		return schema
	}
	return utils.EachDocumentRecord(llmCtx, records, func(dctx context.Context, doc *core.Document) (bool, error) {
		if global.ShuttingDown() {
			return false, errors.New("shutting down")
		}
		if buildExtractionMaterial(doc) == "" {
			return false, nil // skip before touching model config / ORM
		}
		m, err := getLLM()
		if err != nil {
			return false, err
		}
		return processor.extract(dctx, m, getSchema(), doc)
	})
}

func (processor *ExtractEntitiesProcessor) Process(ctx *pipeline.Context) error {
	// record-scope fallback: batch_capable degrades to per-record when nested
	if rec, ok := pipeline.CurrentRecord(ctx); ok {
		llmCtx, cancelFunc := context.WithCancel(ctx.Context)
		defer cancelFunc()
		return utils.EachDocumentRecord(llmCtx, []*event.Event{rec}, func(dctx context.Context, doc *core.Document) (bool, error) {
			if buildExtractionMaterial(doc) == "" {
				return false, nil
			}
			m, err := processor.resolveLLM()
			if err != nil {
				return false, err
			}
			return processor.extract(dctx, m, wiki.ResolveOntologySchema(dctx, processor.resolveKB(dctx, doc)), doc)
		})
	}

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

	llm, err := processor.resolveLLM()
	if err != nil {
		return err
	}
	llmCtx, cancelFunc := context.WithCancel(ctx.Context)
	defer cancelFunc()

	// ontology vocabulary (phase O2): schema-constrained extraction — kb
	// override first, tenant default as fallback; nil means free-form POC
	// list. Resolved per distinct target KB (documents in one batch may
	// belong to different KBs via their datasource bindings)
	schemaByKB := map[string]*wiki.OntologySchemaDoc{}
	resolveSchema := func(dctx context.Context, doc *core.Document) *wiki.OntologySchemaDoc {
		kbID := processor.resolveKB(dctx, doc)
		if schema, ok := schemaByKB[kbID]; ok {
			return schema
		}
		schema := wiki.ResolveOntologySchema(dctx, kbID)
		schemaByKB[kbID] = schema
		return schema
	}

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
			entityIDs, err := processor.extractAndLink(llmCtx, llm, &doc, material, resolveSchema(llmCtx, &doc))
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
// maxContentForExtraction bounds the document text fed to the LLM (WeKnora's
// maxContentForWiki discipline): grounding needs real text, not derivatives.
const maxContentForExtraction = 30000

func buildExtractionMaterial(doc *core.Document) string {
	// extraction must ground on real document text: title/summary/insights
	// are derivatives — on a folder node or an extraction-dead document they
	// are ALL the model sees, and it will happily fabricate entities from a
	// filename (observed: folder docs minting entities from their path)
	if doc.Type == connectors.TypeFolder || strings.TrimSpace(doc.Content) == "" {
		return ""
	}
	content := []rune(doc.Content)
	if len(content) > maxContentForExtraction {
		content = content[:maxContentForExtraction]
	}
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
	parts = append(parts, "Content:\n"+string(content))
	return strings.Join(parts, "\n\n")
}

// resolveKB picks the entity pool for a document: the KB whose datasource
// binding contains the document's source wins; the configured wiki_kb_id
// is only the fallback. A global enrich pipeline serves several KBs, and a
// hardcoded id keeps writing into whatever KB it names (a deleted one, in
// the wild) no matter which datasource the document came from.
func (processor *ExtractEntitiesProcessor) resolveKB(ctx context.Context, doc *core.Document) string {
	if bound := wiki.ResolveKBByDatasource(orm.NewContextWithParent(ctx), doc.Source.ID); bound != "" {
		return bound
	}
	return processor.config.WikiKbID
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
	// pipeline context has no session user: the write flag passes create/update
	// hooks, the read flag passes the search hooks (entity lookups below)
	ormCtx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)

	kbID := processor.resolveKB(ctx, doc)
	// orphan guard: a kb-less write is invisible to every KB-scoped view and
	// silently defeats the whole extraction (live case: 150 described entities
	// stranded while the compile-time promote path showed bare placeholders).
	// Warn loudly — resolution must fail visibly, not quietly.
	if kbID == "" {
		log.Warnf("[%s] document %q (datasource %q) resolved no KB: entities will be stored without kb_id and stay invisible to KB views", ProcessorName, doc.Title, doc.Source.ID)
	}

	resolved := make([]*core.WikiEntity, 0, len(result.Entities))
	seen := map[string]bool{}
	for i := range result.Entities {
		e := &result.Entities[i]
		e.Name = strings.TrimSpace(e.Name)
		if e.Name == "" {
			continue
		}
		entity, created, err := resolveOrCreateEntity(ormCtx, e, doc, processor.config.EntityTypes, schema, kbID)
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
			if kbID != "" {
				if err := createDraftEntityPage(ormCtx, entity, kbID); err != nil {
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

// mergeExtractedInto folds a later extraction into an existing entity:
// new property keys fill in (existing values win — provenance stability
// over churn), missing aliases absorb, and an empty description takes the
// first grounded summary. Without this the update path only appended
// sources and an entity first created without properties/summary stayed
// frozen that way forever (seen live: 12 mentions, 0 properties).
func mergeExtractedInto(existing *core.WikiEntity, extracted *extractedEntity) {
	if existing.Properties == nil {
		existing.Properties = util.MapStr{}
	}
	for k, v := range extracted.Properties {
		if k == "" || v == nil || v == "" {
			continue
		}
		if _, ok := existing.Properties[k]; !ok {
			existing.Properties[k] = v
		}
	}
	for _, a := range normalizeAliases(extracted.Aliases) {
		if a != "" && a != existing.Name && !util.AnyInArrayEquals(existing.Aliases, a) {
			existing.Aliases = append(existing.Aliases, a)
		}
	}
	if strings.TrimSpace(existing.Description) == "" {
		existing.Description = strings.TrimSpace(extracted.Summary)
	}
}

// resolveOrCreateEntity disambiguates by exact name, then alias match
// (case-insensitive retry); unmatched extractions become proposed entities
// with the document recorded as provenance (design doc B2).
func resolveOrCreateEntity(ctx *orm.Context, extracted *extractedEntity, doc *core.Document, allowedTypes []string, schema *wiki.OntologySchemaDoc, kbID string) (*core.WikiEntity, bool, error) {
	orm.WithModel(ctx, &core.WikiEntity{})

	extractedType := strings.ToLower(strings.TrimSpace(extracted.Type))

	if existing := findEntityByName(ctx, kbID, extracted.Name); existing != nil {
		// same name, conflicting types: keep merging the provenance but let
		// the governance queue ask the human whether to split (O2)
		if existing.Type != "" && extractedType != "" && existing.Type != extractedType {
			wiki.FileEntityGovernanceProposal(kbID, existing, core.WikiGovernanceEntityConflict,
				fmt.Sprintf("document %q mentions %q as %q, existing entity is %q", doc.Title, extracted.Name, extractedType, existing.Type),
				util.MapStr{"doc_id": doc.ID, "existing_type": existing.Type, "extracted_type": extractedType})
		}
		mergeExtractedInto(existing, extracted)
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
		if existing := findEntityByName(ctx, kbID, alias); existing != nil {
			if existing.Type != "" && extractedType != "" && existing.Type != extractedType {
				wiki.FileEntityGovernanceProposal(kbID, existing, core.WikiGovernanceEntityConflict,
				fmt.Sprintf("document %q mentions alias %q of %q as %q, existing entity is %q", doc.Title, alias, extracted.Name, extractedType, existing.Type),
				util.MapStr{"doc_id": doc.ID, "alias": alias, "existing_type": existing.Type, "extracted_type": extractedType})
		}
		mergeExtractedInto(existing, extracted)
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
	entityType := canonicalizeEntityType(extractedType, allowedTypes, schema)

	status := core.WikiEntityProposed // human review required (D1)
	// auto-approve KB: new extractions publish directly — the opt-in
	// per-KB toggle replaces the review gate for this source
	if wiki.KBAutoApprove(ctx, kbID) {
		status = core.WikiEntityPublished
	}
	entity := &core.WikiEntity{
		// the KB scope is the whole point: without it the entity pool is
		// orphaned (kb_id-less rows no KB-scoped query can ever see) —
		// exactly how the Books KB's entity tab stayed empty while
		// extraction kept "working"
		KbID:        kbID,
		Type:        entityType,
		Name:        extracted.Name,
		Aliases:     normalizeAliases(extracted.Aliases),
		Properties:  extracted.Properties,
		Description: strings.TrimSpace(extracted.Summary),
		Status:      status,
		Sources:     []core.WikiSourceReference{documentSourceRef(doc, extracted.Evidence)},
	}
	if schema != nil {
		if dropped := wiki.SanitizeEntityAgainstSchema(entity, schema); len(dropped) > 0 {
			// Warn, not Debug: silent drops made declared properties vanish
			// invisibly (live data showed 0/363 entities with properties).
			log.Warnf("[%s] schema sanitize dropped from %q: %v", ProcessorName, extracted.Name, dropped)
		}
	}
	// name-anchored id: extraction resolves identity by name, so two
	// workers racing past the lookup write the same row instead of leaving
	// twin proposed entities that never self-merge; the loser's overwrite
	// differs only in provenance, and the next pass re-appends sources
	entity.SetID(core.AnchoredEntityID(kbID, entity.Name))
	if err := orm.Create(ctx, entity); err != nil {
		// a concurrent extraction created the row first (409) — the design
		// wants one row with merged provenance, so fall back to the merge
		// path on the winner instead of failing the batch
		if strings.Contains(err.Error(), "version_conflict") {
			if winner := findEntityByName(ctx, kbID, entity.Name); winner != nil {
				mergeExtractedInto(winner, extracted)
				appendEntitySource(winner, extracted, doc)
				if uerr := orm.Update(ctx, winner); uerr != nil {
					return nil, false, uerr
				}
				return winner, false, nil
			}
		}
		return nil, false, err
	}
	return entity, true, nil
}

// canonicalizeEntityType matches the LLM-extracted type against the
// allowed vocabulary case-insensitively and returns the DECLARED casing
// ("Organization" from the model → "organization" from the schema).
// Unknown types degrade to "concept" instead of dropping.
func canonicalizeEntityType(extracted string, allowedTypes []string, schema *wiki.OntologySchemaDoc) string {
	if extracted == "" {
		return "concept"
	}
	for _, allowed := range allowedTypes {
		if strings.EqualFold(allowed, extracted) {
			return allowed // declared casing wins
		}
	}
	return "concept" // unknown vocabulary: degrade instead of dropping
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
		Content:     wiki.DraftEntityContentFor(ctx, kbID, entity),
		Sources:     entity.Sources, // provenance: every claim traces to its source doc
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
func findEntityByName(ctx *orm.Context, kbID, name string) *core.WikiEntity {
	return wiki.FindEntityByNameOrAlias(ctx, name, "", kbID)
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
	ref := documentSourceRef(doc, extracted.Evidence)
	for i := range entity.Sources {
		if entity.Sources[i].DocID == doc.ID {
			// re-ingestion: refresh the block locator in place (older refs
			// predate chunk matching) without stacking a duplicate
			if entity.Sources[i].Locator == "" && ref.Locator != "" {
				entity.Sources[i].Locator = ref.Locator
			}
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
	entity.Sources = append(entity.Sources, ref)
}

func documentSourceRef(doc *core.Document, evidence string) core.WikiSourceReference {
	ref := core.WikiSourceReference{
		DocID:   doc.ID,
		Title:   doc.Title,
		Excerpt: evidence,
		// block-level citation (WeKnora-style): locate the evidence quote
		// inside the document's chunks so the wiki citation can point at
		// the exact block, not just the document
		Locator: chunkLocator(doc, evidence),
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

// chunkLocator matches the LLM's evidence quote against the document's
// chunks — whitespace-insensitive on the full quote, falling back to its
// distinctive head fragment (LLMs paraphrase around the edges) — and
// returns the chunk's breadcrumb, or its ordinal when the chunk carries
// no breadcrumb. Empty when nothing matches or the doc has no chunks.
func chunkLocator(doc *core.Document, evidence string) string {
	if len(doc.Chunks) == 0 || strings.TrimSpace(evidence) == "" {
		return ""
	}
	normalize := func(s string) string { return strings.Join(strings.Fields(s), "") }
	ev := normalize(evidence)
	if ev == "" {
		return ""
	}
	head := ev
	if r := []rune(head); len(r) > 24 {
		head = string(r[:24])
	}
	for i := range doc.Chunks {
		ct := normalize(doc.Chunks[i].Text)
		if ct == "" {
			continue
		}
		if strings.Contains(ct, ev) || strings.Contains(ct, normalize(head)) {
			if doc.Chunks[i].Breadcrumb != "" {
				return doc.Chunks[i].Breadcrumb
			}
			return fmt.Sprintf("块 %d/%d", i+1, len(doc.Chunks))
		}
	}
	return ""
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
	llmCtx, llmCancel := utils.PipelineLLMContext(ctx)
	defer llmCancel()
	ctx = llmCtx
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
			"- An entity MUST be a proper noun naming a specific thing (a product, organization, person, named document, named feature/mechanism). Skip generic nouns, verbs, adjectives\n"+
			"- Do NOT extract: configuration keys or parameter names (e.g. \"include_private_book\", \"max_retries\"), UI labels or menu items, HTTP/API field names, file paths, code identifiers, error messages, plural/common words (e.g. \"name\", \"source\", \"config\", \"share\", \"payload\")\n"+
			"- When unsure whether something is a nameable entity or just common wording, skip it — a smaller high-precision pool beats a large noisy one\n"+
			"- Use one of the allowed entity types for \"type\"\n"+
			"- \"aliases\": alternate names/s spellings found in the text (may be empty)\n"+
			"- \"properties\": up to 5 short factual attributes as {\"key\": \"value\"}\n"+
			"- \"relations\": links to OTHER entities extracted from the same document as {\"relation\": \"verb_phrase\", \"target\": \"<entity name>\"}\n"+
			"- \"evidence\": a VERBATIM excerpt of 15-80 characters copied character-for-character from the document text where the entity is mentioned - same language as the document, no translation, no paraphrase, no added words (it is matched back against the source text to locate the exact block)\n"+
			"- \"summary\": ONE sentence, grounded strictly in the document, saying what this entity is (identity/category plus the most distinguishing fact). 15-60 characters, same language as the document, no speculation\n"+
			"- Names and aliases MUST be in their original language as they appear\n%s\n"+
			"Format:\n"+
			`{"entities":[{"name":"","type":"","aliases":[],"properties":{},"relations":[{"relation":"","target":""}],"evidence":"","summary":""}]}`+"\n\n"+
			"Document:\n%s\n\n"+
			"Generate the JSON object now.",
		vocabulary,
		config.MaxEntities,
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
		if len(t.Description) > 0 {
			fmt.Fprintf(&b, ": %s", t.Description)
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
	payload := trimmed[jsonStart : jsonEnd+1]
	result := extractionResult{}
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		// live failure mode (~14% of one GLM run): slightly malformed JSON —
		// trailing commas, missing commas between adjacent objects. Repair
		// the common cases once before giving up; the doc-level retry costs
		// a full LLM round trip.
		repaired := repairLooseJSON(payload)
		// the other live failure mode: verbatim evidence copies raw markdown
		// into string values with unescaped quotes/newlines
		sanitized := repairLooseJSON(sanitizeRawStrings(payload))
		// the dominant structural failure: {"relation":..} objects dropped
		// at entity-object key positions instead of inside a relations array
		hoisted := hoistStrayRelationObjects(sanitized)
		candidates := []string{repaired, sanitized, hoisted}
		// models sometimes double-wrap the object ({{...}}) or answer with
		// a bare array — peel/wrap one layer each before conceding
		peeled := strings.TrimSpace(hoisted)
		if strings.HasPrefix(peeled, "{{") && strings.HasSuffix(peeled, "}}") {
			candidates = append(candidates, peeled[1:len(peeled)-1])
		}
		if strings.HasPrefix(peeled, "[") {
			candidates = append(candidates, `{"entities":`+peeled+`}`)
		}
		parsed := false
		for _, cand := range candidates {
			if rerr := json.Unmarshal([]byte(cand), &result); rerr == nil {
				parsed = true
				break
			}
		}
		if !parsed {
			log.Warnf("[%s] unparseable extraction response (head): %.300s", ProcessorName, payload)
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
		log.Warnf("[%s] recovered malformed JSON from extraction response via repair", ProcessorName)
	}
	return &result, nil
}

// hoistStrayRelationObjects repairs the dominant live structural failure:
// the model drops {"relation":..,"target":..} objects at entity-object KEY
// positions ("properties":{...},{"relation":...}]) instead of inside a
// relations array — strict unmarshal dies with "invalid character '{'
// looking for beginning of object key string". Each stray object is
// gathered and re-attached as a relations array on its owning object.
func hoistStrayRelationObjects(s string) string {
	if !strings.Contains(s, `"relation"`) {
		return s
	}
	// matching close-brace index for the object opening at `open`
	matchObj := func(open int) int {
		depth := 0
		inStr, escape := false, false
		for i := open; i < len(s); i++ {
			c := s[i]
			if inStr {
				if escape {
					escape = false
				} else if c == '\\' {
					escape = true
				} else if c == '"' {
					inStr = false
				}
				continue
			}
			switch c {
			case '"':
				inStr = true
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return i
				}
			}
		}
		return len(s) - 1
	}

	type frame struct {
		isObj  bool
		keyPos bool
		rels   []string
		hasArr bool // the object already declares a real "relations" key
	}
	var b strings.Builder
	var stack []*frame
	inStr, escape := false, false
	keyStart := -1
	lastByte := byte(0) // last byte actually written (never whitespace)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			b.WriteByte(c)
			lastByte = c
			if escape {
				escape = false
			} else if c == '\\' {
				escape = true
			} else if c == '"' {
				inStr = false
				if len(stack) > 0 && keyStart >= 0 {
					if f := stack[len(stack)-1]; f.isObj && s[keyStart:i+1] == `"relations"` {
						f.hasArr = true
					}
				}
				keyStart = -1
			}
			continue
		}
		switch c {
		case '"':
			b.WriteByte(c)
			inStr = true
			keyStart = -1
			if len(stack) > 0 {
				if f := stack[len(stack)-1]; f.isObj && f.keyPos {
					keyStart = i
				}
			}
		case '{':
			if len(stack) > 0 {
				if f := stack[len(stack)-1]; f.isObj && f.keyPos {
					// stray value object at key position: skip it, keep it
					// if it looks like a relation pair
					j := matchObj(i)
					obj := s[i : j+1]
					if strings.Contains(obj, `"relation"`) {
						f.rels = append(f.rels, obj)
					}
					i = j
					continue
				}
			}
			stack = append(stack, &frame{isObj: true, keyPos: true})
			b.WriteByte(c)
		case '[':
			stack = append(stack, &frame{})
			b.WriteByte(c)
		case '}':
			f := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if f != nil && f.isObj && len(f.rels) > 0 && !f.hasArr {
				// the stray's separating comma is already in the output —
				// don't double it
				if lastByte == ',' {
					b.WriteString(`"relations":[` + strings.Join(f.rels, ",") + `]`)
				} else {
					b.WriteString(`,"relations":[` + strings.Join(f.rels, ",") + `]`)
				}
			}
			b.WriteByte(c)
			lastByte = c
		case ']':
			// mismatch guard: models sometimes omit the entity object's
			// closing brace before the array close — synthesize the
			// missing object closes (attaching hoisted relations) until
			// the matching array frame is on top
			for len(stack) > 0 {
				f := stack[len(stack)-1]
				if !f.isObj {
					stack = stack[:len(stack)-1]
					break
				}
				stack = stack[:len(stack)-1]
				if len(f.rels) > 0 && !f.hasArr {
					if lastByte == ',' {
						b.WriteString(`"relations":[` + strings.Join(f.rels, ",") + `]`)
					} else {
						b.WriteString(`,"relations":[` + strings.Join(f.rels, ",") + `]`)
					}
				}
				b.WriteByte('}')
				lastByte = '}'
			}
			b.WriteByte(c)
			lastByte = c
		case ':':
			b.WriteByte(c)
			// a value follows — the next { is the value, not a stray
			if len(stack) > 0 {
				if f := stack[len(stack)-1]; f.isObj {
					f.keyPos = false
				}
			}
		case ',':
			b.WriteByte(c)
			if len(stack) > 0 {
				if f := stack[len(stack)-1]; f.isObj {
					f.keyPos = true
				}
			}
		default:
			b.WriteByte(c)
		}
		lastByte = c
	}
	return b.String()
}

// sanitizeRawStrings escapes raw quotes/newlines/tabs inside JSON string
// values: the "evidence" field copies document text verbatim, so models
// routinely emit unescaped `"` / newlines there and strict unmarshal dies.
// A closing quote is only accepted when the next non-space byte is
// structural (, } ] : or end) — anything else is an inner raw quote.
func sanitizeRawStrings(s string) string {
	var b strings.Builder
	inStr := false
	escape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !inStr {
			b.WriteByte(c)
			if c == '"' {
				inStr = true
			}
			continue
		}
		if escape {
			b.WriteByte(c)
			escape = false
			continue
		}
		switch c {
		case '\\':
			b.WriteByte(c)
			escape = true
		case '"':
			j := i + 1
			for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
				j++
			}
			if j >= len(s) || s[j] == ',' || s[j] == '}' || s[j] == ']' || s[j] == ':' {
				b.WriteByte(c)
				inStr = false
			} else {
				b.WriteString("\\\"")
			}
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// repairLooseJSON fixes two common LLM JSON defects: trailing commas
// (`,}` / `,]`) and missing commas between adjacent values (`}{`, `]["`,
// `"a" "b"`). The missing-comma pass runs a small string-aware scanner so
// quotes containing those byte pairs are never touched.
func repairLooseJSON(s string) string {
	// trailing commas
	trailing := regexp.MustCompile(`,\s*([}\]])`)
	s = trailing.ReplaceAllString(s, "$1")

	// missing commas between adjacent elements
	var b strings.Builder
	prev := byte(0) // last structural byte outside strings
	inStr := false
	escape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			b.WriteByte(c)
			switch {
			case escape:
				escape = false
			case c == '\\':
				escape = true
			case c == '"':
				inStr = false
				prev = '"'
			}
			continue
		}
		switch c {
		case '"':
			// a value straight after another value needs a comma first
			if prev == '}' || prev == ']' || prev == '"' {
				b.WriteByte(',')
			}
			inStr = true
		case '{', '[':
			if prev == '}' || prev == ']' || prev == '"' {
				b.WriteByte(',')
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 't', 'f', 'n':
			// number/true/false/null literal starting right after a value
			if prev == '}' || prev == ']' || prev == '"' {
				b.WriteByte(',')
			}
		}
		b.WriteByte(c)
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			// whitespace never resets the terminator state: `} {` must
			// still be seen as adjacent values
			continue
		}
		switch c {
		case '}', ']', ',', ':':
			prev = c
		default:
			// openers and literal content are not value terminators; using
			// a non-triggering sentinel also keeps multi-char literals
			// (123, true, null) from emitting a comma mid-token
			prev = 0
		}
	}
	return b.String()
}
