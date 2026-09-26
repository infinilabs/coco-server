/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	log "github.com/cihub/seelog"

	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

/* Ontology vocabulary layer (design doc §3.5, phase O1): the schema that
 * makes the wiki "ontology-based" instead of free-form tagging. One schema
 * document per scope — "tenant" globally, "kb:<id>" as an override — holds
 * the declared entity types, their typed properties and the relation
 * vocabulary. Entity writes validate against the resolved schema (kb scope
 * first, tenant fallback); with no schema on file validation stays lenient
 * so existing data and tests are unaffected.
 */

const (
	ontologyTenantScope = "tenant"
	ontologyKBScopeFmt  = "kb:%s"

	// property value types
	ontologyPropString = "string"
	ontologyPropText   = "text"
	ontologyPropNumber = "number"
	ontologyPropBool   = "boolean"
	ontologyPropDate   = "date"
	ontologyPropURL    = "url"
	ontologyPropEnum   = "enum"

	// relation cardinality
	ontologyCardinalityOne = "one"
)

// OntologyPropertyDef declares one typed attribute of an entity type.
type OntologyPropertyDef struct {
	Key      string   `json:"key"`
	Label    string   `json:"label,omitempty"`
	Type     string   `json:"type"` // string | text | number | boolean | date | url | enum
	Required bool     `json:"required,omitempty"`
	Enum     []string `json:"enum,omitempty"` // allowed values when type=enum
}

// OntologyRelationDef declares one outgoing relation of an entity type.
type OntologyRelationDef struct {
	Name        string `json:"name"`
	Label       string `json:"label,omitempty"`
	TargetType  string `json:"target_type"`           // declared entity type, or "*" for any
	Cardinality string `json:"cardinality,omitempty"` // "one" | "many" (default many)
	Inverse     string `json:"inverse,omitempty"`     // relation name materialized on the target side
}

// OntologyEntityTypeDef declares one entity type.
type OntologyEntityTypeDef struct {
	Name       string                `json:"name"`
	Label      string                `json:"label,omitempty"`
	Icon       string                `json:"icon,omitempty"`
	Properties []OntologyPropertyDef `json:"properties,omitempty"`
	Relations  []OntologyRelationDef `json:"relations,omitempty"`
}

// OntologySchemaDoc is the stored schema document.
type OntologySchemaDoc struct {
	EntityTypes []OntologyEntityTypeDef `json:"entity_types"`
}

// normalize fills defaults and rejects self-contradictory declarations.
func (doc *OntologySchemaDoc) normalize() error {
	if doc == nil {
		return nil
	}
	seen := map[string]bool{}
	for i := range doc.EntityTypes {
		t := &doc.EntityTypes[i]
		t.Name = strings.TrimSpace(t.Name)
		if t.Name == "" {
			return fmt.Errorf("entity_types[%d].name is required", i)
		}
		if seen[t.Name] {
			return fmt.Errorf("entity type %q declared twice", t.Name)
		}
		seen[t.Name] = true

		props := map[string]bool{}
		for j := range t.Properties {
			p := &t.Properties[j]
			p.Key = strings.TrimSpace(p.Key)
			if p.Key == "" {
				return fmt.Errorf("entity type %q: properties[%d].key is required", t.Name, j)
			}
			if props[p.Key] {
				return fmt.Errorf("entity type %q: property %q declared twice", t.Name, p.Key)
			}
			props[p.Key] = true
			switch p.Type {
			case "":
				p.Type = ontologyPropString
			case ontologyPropString, ontologyPropText, ontologyPropNumber, ontologyPropBool,
				ontologyPropDate, ontologyPropURL, ontologyPropEnum:
			default:
				return fmt.Errorf("entity type %q: property %q has unknown type %q", t.Name, p.Key, p.Type)
			}
			if p.Type == ontologyPropEnum && len(p.Enum) == 0 {
				return fmt.Errorf("entity type %q: enum property %q needs at least one allowed value", t.Name, p.Key)
			}
		}

		rels := map[string]bool{}
		for j := range t.Relations {
			r := &t.Relations[j]
			r.Name = strings.TrimSpace(r.Name)
			if r.Name == "" {
				return fmt.Errorf("entity type %q: relations[%d].name is required", t.Name, j)
			}
			if rels[r.Name] {
				return fmt.Errorf("entity type %q: relation %q declared twice", t.Name, r.Name)
			}
			rels[r.Name] = true
			if r.TargetType == "" {
				r.TargetType = "*"
			}
			if r.Cardinality != "" && r.Cardinality != ontologyCardinalityOne && r.Cardinality != "many" {
				return fmt.Errorf("entity type %q: relation %q has unknown cardinality %q", t.Name, r.Name, r.Cardinality)
			}
		}
	}
	return nil
}

func (doc *OntologySchemaDoc) typeDef(name string) *OntologyEntityTypeDef {
	if doc == nil {
		return nil
	}
	for i := range doc.EntityTypes {
		if doc.EntityTypes[i].Name == name {
			return &doc.EntityTypes[i]
		}
	}
	return nil
}

/* ---------------- storage ---------------- */

// loadOntologySchemaForKB resolves kb:<id> first, then the tenant schema.
func loadOntologySchemaForKB(ctx context.Context, kbID string) *OntologySchemaDoc {
	if kbID != "" {
		if doc := loadOntologySchema(ctx, fmt.Sprintf(ontologyKBScopeFmt, kbID)); doc != nil {
			return doc
		}
	}
	return loadOntologySchema(ctx, ontologyTenantScope)
}

func loadOntologySchema(ctx context.Context, scope string) *OntologySchemaDoc {
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ormCtx, &core.WikiOntologySchema{})

	builder := orm.NewQuery().Size(1).Filter(orm.TermQuery("scope", scope))
	res, err := orm.SearchV2(ormCtx, builder)
	if err != nil {
		return nil
	}
	docs, _, err := elastic.DecodeHits[core.WikiOntologySchema](res)
	if err != nil || len(docs) == 0 || docs[0].Schema == nil {
		return nil
	}
	raw, err := util.ToJSONBytes(docs[0].Schema)
	if err != nil {
		return nil
	}
	var doc OntologySchemaDoc
	if err := util.FromJSONBytes(raw, &doc); err != nil || len(doc.EntityTypes) == 0 {
		return nil
	}
	return &doc
}

// saveOntologySchema upserts the schema document of one scope.
func saveOntologySchema(ctx context.Context, scope string, doc *OntologySchemaDoc) error {
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ormCtx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ormCtx.Refresh = orm.WaitForRefresh
	orm.WithModel(ormCtx, &core.WikiOntologySchema{})

	builder := orm.NewQuery().Size(1).Filter(orm.TermQuery("scope", scope))
	res, err := orm.SearchV2(ormCtx, builder)
	if err != nil {
		return err
	}
	existing, _, err := elastic.DecodeHits[core.WikiOntologySchema](res)
	if err != nil {
		return err
	}

	schemaMap := map[string]interface{}{}
	if raw, err := util.ToJSONBytes(doc); err == nil {
		_ = util.FromJSONBytes(raw, &schemaMap)
	}
	obj := &core.WikiOntologySchema{Scope: scope, Schema: schemaMap}
	if len(existing) > 0 {
		obj = &existing[0]
		obj.Schema = schemaMap
		return orm.Update(ormCtx, obj)
	}
	return orm.Create(ormCtx, obj)
}

/* ---------------- validation ---------------- */

// validateEntityAgainstSchema enforces the tenant vocabulary on entity writes.
// Without a schema (or a type not covered by one) validation is lenient:
// the ontology constrains what it knows about, it never blocks legacy or
// exploratory data.
func validateEntityAgainstSchema(ctx context.Context, entity *core.WikiEntity) error {
	return validateEntityWithDoc(ctx, entity, loadOntologySchema(ctx, ontologyTenantScope))
}

// validateEntityForKB enforces the KB-scoped vocabulary: kb:<id> override
// first, tenant schema as the default. Interactive creation from a KB
// context passes the kb_id through so KB-specific vocabularies are usable
// in the UI (schema ownership fix, W3).
func validateEntityForKB(ctx context.Context, entity *core.WikiEntity, kbID string) error {
	return validateEntityWithDoc(ctx, entity, loadOntologySchemaForKB(ctx, kbID))
}

func validateEntityWithDoc(ctx context.Context, entity *core.WikiEntity, doc *OntologySchemaDoc) error {
	if doc == nil {
		return nil
	}
	if entity.Type == "" {
		return nil // untyped entities predate the vocabulary
	}
	typeDef := doc.typeDef(entity.Type)
	if typeDef == nil {
		return fmt.Errorf("entity type %q is not declared in the ontology schema", entity.Type)
	}

	for _, prop := range typeDef.Properties {
		value, present := entity.Properties[prop.Key]
		if !present || isEmptyOntologyValue(value) {
			if prop.Required {
				return fmt.Errorf("property %q of entity type %q is required", prop.Key, typeDef.Name)
			}
			continue
		}
		if err := checkOntologyValueType(prop, value); err != nil {
			return fmt.Errorf("property %q of entity %q: %v", prop.Key, entity.Name, err)
		}
	}

	seenRelationCount := map[string]int{}
	for _, rel := range entity.Relations {
		seenRelationCount[rel.Relation]++

		declared := findRelationDef(typeDef, rel.Relation)
		if declared == nil {
			return fmt.Errorf("relation %q is not declared for entity type %q", rel.Relation, typeDef.Name)
		}
		if declared.Cardinality == ontologyCardinalityOne && seenRelationCount[rel.Relation] > 1 {
			return fmt.Errorf("relation %q of entity type %q allows at most one target", rel.Relation, typeDef.Name)
		}
		if declared.TargetType != "*" && rel.TargetID != "" {
			if targetType := entityTypeByID(ctx, rel.TargetID); targetType != "" && targetType != declared.TargetType {
				return fmt.Errorf("relation %q requires target type %q, got %q", rel.Relation, declared.TargetType, targetType)
			}
		}
	}
	return nil
}

func findRelationDef(typeDef *OntologyEntityTypeDef, name string) *OntologyRelationDef {
	for i := range typeDef.Relations {
		if typeDef.Relations[i].Name == name {
			return &typeDef.Relations[i]
		}
	}
	return nil
}

// entityTypeByID resolves an entity's type for relation target checks;
// unresolvable targets are skipped (they may be created in the same batch).
func entityTypeByID(ctx context.Context, id string) string {
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ormCtx, &core.WikiEntity{})

	var target core.WikiEntity
	target.SetID(id)
	exists, err := orm.GetV2(ormCtx, &target)
	if err != nil || !exists {
		return ""
	}
	return target.Type
}

func isEmptyOntologyValue(v interface{}) bool {
	switch val := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(val) == ""
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	}
	return false
}

func checkOntologyValueType(prop OntologyPropertyDef, value interface{}) error {
	switch prop.Type {
	case ontologyPropEnum:
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected a string value")
		}
		for _, allowed := range prop.Enum {
			if s == allowed {
				return nil
			}
		}
		return fmt.Errorf("value %q is not one of the allowed enum values", s)
	case ontologyPropNumber:
		switch value.(type) {
		case float64, float32, int, int64, int32:
			return nil
		}
		return fmt.Errorf("expected a number")
	case ontologyPropBool:
		switch value.(type) {
		case bool:
			return nil
		}
		return fmt.Errorf("expected a boolean")
	default: // string | text | date | url — JSON strings
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected a string value")
		}
	}
	return nil
}

/* ---------------- seed ---------------- */

// defaultOntologySchema is a pragmatic general-purpose vocabulary: enough
// structure for enterprise knowledge without pretending to be OWL.
func defaultOntologySchema() *OntologySchemaDoc {
	return &OntologySchemaDoc{
		EntityTypes: []OntologyEntityTypeDef{
			{
				Name: "person", Label: "人员", Icon: "👤",
				Properties: []OntologyPropertyDef{
					{Key: "email", Label: "邮箱", Type: ontologyPropString},
					{Key: "title", Label: "职位", Type: ontologyPropString},
					{Key: "department", Label: "部门", Type: ontologyPropString},
				},
				Relations: []OntologyRelationDef{
					{Name: "works_for", Label: "任职于", TargetType: "organization", Inverse: "has_employee"},
					{Name: "owns", Label: "负责", TargetType: "*", Inverse: "owned_by"},
				},
			},
			{
				Name: "organization", Label: "组织", Icon: "🏢",
				Properties: []OntologyPropertyDef{
					{Key: "website", Label: "网站", Type: ontologyPropURL},
					{Key: "industry", Label: "行业", Type: ontologyPropString},
				},
				Relations: []OntologyRelationDef{
					{Name: "has_employee", Label: "成员", TargetType: "person", Inverse: "works_for"},
					{Name: "part_of", Label: "隶属于", TargetType: "organization", Cardinality: ontologyCardinalityOne, Inverse: "has_part"},
					{Name: "has_part", Label: "下级组织", TargetType: "organization", Inverse: "part_of"},
				},
			},
			{
				Name: "product", Label: "产品", Icon: "📦",
				Properties: []OntologyPropertyDef{
					{Key: "version", Label: "版本", Type: ontologyPropString},
					{Key: "status", Label: "状态", Type: ontologyPropEnum, Enum: []string{"active", "deprecated", "beta"}},
					{Key: "released_at", Label: "发布日期", Type: ontologyPropDate},
				},
				Relations: []OntologyRelationDef{
					{Name: "made_by", Label: "生产方", TargetType: "organization", Cardinality: ontologyCardinalityOne, Inverse: "makes"},
					{Name: "makes", Label: "生产", TargetType: "product", Inverse: "made_by"},
					{Name: "depends_on", Label: "依赖", TargetType: "product"},
				},
			},
			{
				Name: "concept", Label: "概念", Icon: "💡",
				Properties: []OntologyPropertyDef{
					{Key: "aliases_hint", Label: "别名提示", Type: ontologyPropText},
				},
				Relations: []OntologyRelationDef{
					{Name: "relates_to", Label: "相关", TargetType: "*"},
					{Name: "subclass_of", Label: "上位概念", TargetType: "concept", Cardinality: ontologyCardinalityOne},
				},
			},
			{
				Name: "event", Label: "事件", Icon: "📅",
				Properties: []OntologyPropertyDef{
					{Key: "happened_at", Label: "发生时间", Type: ontologyPropDate, Required: true},
					{Key: "location", Label: "地点", Type: ontologyPropString},
				},
				Relations: []OntologyRelationDef{
					{Name: "involves", Label: "涉及", TargetType: "*"},
					{Name: "hosted_by", Label: "主办", TargetType: "organization"},
				},
			},
			{
				Name: "document", Label: "文档", Icon: "📄",
				Properties: []OntologyPropertyDef{
					{Key: "url", Label: "链接", Type: ontologyPropURL},
					{Key: "published_at", Label: "发布日期", Type: ontologyPropDate},
				},
				Relations: []OntologyRelationDef{
					{Name: "authored_by", Label: "作者", TargetType: "person"},
					{Name: "mentions", Label: "提及", TargetType: "*", Inverse: "mentioned_in"},
					{Name: "mentioned_in", Label: "被提及于", TargetType: "*"},
				},
			},
		},
	}
}

// seedDefaultOntologySchema inserts the tenant schema once; local edits are
// never overwritten (same contract as skill/assistant-template seeds).
func seedDefaultOntologySchema() {
	ctx := context.Background()
	if loadOntologySchema(ctx, ontologyTenantScope) != nil {
		return
	}
	doc := defaultOntologySchema()
	if err := doc.normalize(); err != nil {
		log.Errorf("wiki: default ontology schema invalid: %v", err)
		return
	}
	if err := saveOntologySchema(ctx, ontologyTenantScope, doc); err != nil {
		log.Errorf("wiki: seeding default ontology schema failed: %v", err)
		return
	}
	log.Infof("wiki: seeded default ontology schema (%d entity types)", len(doc.EntityTypes))
}

func scheduleSeedOntologySchema() {
	global.RegisterFuncAfterSetup(func() {
		seedDefaultOntologySchema()
	})
}

/* ---------------- GET/PUT /wiki/ontology/schema ---------------- */

type ontologySchemaRequest struct {
	KbID   string             `json:"kb_id"`
	Schema *OntologySchemaDoc `json:"schema"`
}

// getOntologySchema returns the schema a KB resolves to (kb override first,
// tenant fallback) — the vocabulary entities of that context validate
// against. An empty entity_types list means "no schema on file".
func (h *APIHandler) getOntologySchema(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	kbID := req.URL.Query().Get("kb")
	doc := loadOntologySchemaForKB(req.Context(), kbID)
	if doc == nil {
		doc = &OntologySchemaDoc{}
	}
	h.WriteGetOKJSON(w, "schema", util.MapStr{
		"kb_id":        kbID,
		"entity_types": doc.EntityTypes,
	})
}

// putOntologySchema replaces the schema document of one scope (tenant, or
// kb:<id> when kb_id is set). The document is normalized before saving so
// duplicate types/properties or unknown value types cannot slip in.
func (h *APIHandler) putOntologySchema(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body := ontologySchemaRequest{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.Error400(w, err.Error())
		return
	}
	if body.Schema == nil {
		h.Error400(w, "schema is required")
		return
	}
	if err := body.Schema.normalize(); err != nil {
		h.Error400(w, err.Error())
		return
	}

	scope := ontologyTenantScope
	if body.KbID != "" {
		ormCtx := orm.NewContextWithParent(req.Context())
		ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
		orm.WithModel(ormCtx, &core.WikiKnowledgeBase{})

		var kb core.WikiKnowledgeBase
		kb.SetID(body.KbID)
		exists, err := orm.GetV2(ormCtx, &kb)
		if err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !exists {
			h.WriteOpRecordNotFoundJSON(w, body.KbID)
			return
		}
		scope = fmt.Sprintf(ontologyKBScopeFmt, body.KbID)
	}

	if err := saveOntologySchema(req.Context(), scope, body.Schema); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteUpdatedOKJSON(w, scope)
}

/* ---------------- exports for the extraction pipeline ---------------- */

// ResolveOntologySchema is the pipeline-facing resolver: kb-scoped override
// first, tenant schema as the default vocabulary.
func ResolveOntologySchema(ctx context.Context, kbID string) *OntologySchemaDoc {
	return loadOntologySchemaForKB(ctx, kbID)
}

// SanitizeEntityAgainstSchema strips what the vocabulary rejects from a
// pipeline-produced entity before it is persisted: undeclared relations,
// cardinality-one overflow and declared properties whose value type does not
// conform. Dropped items are reported as human-readable strings for logging.
// Untyped entities or types missing from the schema pass through untouched
// (lenient by design — extraction proposes, the vocabulary constrains what
// it knows about).
func SanitizeEntityAgainstSchema(entity *core.WikiEntity, doc *OntologySchemaDoc) []string {
	if doc == nil || entity == nil {
		return nil
	}
	typeDef := doc.typeDef(entity.Type)
	if typeDef == nil {
		return nil
	}

	var dropped []string

	keptProps := make(map[string]interface{}, len(entity.Properties))
	for key, value := range entity.Properties {
		def := findPropertyDef(typeDef, key)
		if def == nil {
			keptProps[key] = value // free-form attributes survive (lenient)
			continue
		}
		if err := checkOntologyValueType(*def, value); err != nil {
			dropped = append(dropped, fmt.Sprintf("property %s: %v", key, err))
			continue
		}
		keptProps[key] = value
	}
	entity.Properties = keptProps

	keptRels := make([]core.WikiEntityRelation, 0, len(entity.Relations))
	seenCardinality := map[string]bool{}
	for _, rel := range entity.Relations {
		def := findRelationDef(typeDef, rel.Relation)
		if def == nil {
			dropped = append(dropped, fmt.Sprintf("relation %s: not declared for type %s", rel.Relation, typeDef.Name))
			continue
		}
		if def.Cardinality == ontologyCardinalityOne {
			if seenCardinality[rel.Relation] {
				dropped = append(dropped, fmt.Sprintf("relation %s: cardinality one, extra target dropped", rel.Relation))
				continue
			}
			seenCardinality[rel.Relation] = true
		}
		if def.TargetType != "*" && rel.TargetID != "" {
			if targetType := entityTypeByID(context.Background(), rel.TargetID); targetType != "" && targetType != def.TargetType {
				dropped = append(dropped, fmt.Sprintf("relation %s: target type %s != %s", rel.Relation, targetType, def.TargetType))
				continue
			}
		}
		keptRels = append(keptRels, rel)
	}
	entity.Relations = keptRels
	return dropped
}

func findPropertyDef(typeDef *OntologyEntityTypeDef, key string) *OntologyPropertyDef {
	for i := range typeDef.Properties {
		if typeDef.Properties[i].Key == key {
			return &typeDef.Properties[i]
		}
	}
	return nil
}

// FileEntityGovernanceProposal files an entity-dimension governance proposal
// (extraction conflicts and duplicates). Idempotent per open (entity, type);
// the KB owner gets a notification. Best-effort — returns false when the
// proposal could not be recorded.
func FileEntityGovernanceProposal(kbID string, entity *core.WikiEntity, pType, reason string, evidence util.MapStr) bool {
	if entity == nil || entity.ID == "" {
		return false
	}
	ctx := context.Background()
	open := openProposalKeys(ctx)
	key := entity.ID + "|" + pType
	if open[key] {
		return false
	}

	wctx := orm.NewContext()
	wctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	wctx.Refresh = orm.WaitForRefresh
	orm.WithModel(wctx, &core.WikiGovernanceProposal{})

	proposal := &core.WikiGovernanceProposal{
		KbID:         kbID,
		ArticleID:    entity.ID,
		ArticleTitle: entity.Name,
		Type:         pType,
		Status:       core.WikiGovernanceOpen,
		Reason:       reason,
		Evidence:     evidence,
	}
	if err := orm.Create(wctx, proposal); err != nil {
		log.Warnf("wiki: entity governance proposal create failed: %v", err)
		return false
	}
	notifyOwner(entity.GetOwnerID(), "kb", kbID, "governance",
		fmt.Sprintf("Governance: %s — entity %q (%s)", governanceTypeLabel(pType), entity.Name, reason))
	return true
}

/* ---------------- POST /wiki/article/:id/_relink ---------------- */

// relinkArticle re-resolves the article's wikilinks against the (possibly
// just-extended) entity set — the companion of the dangling-link repair
// flow: after proposed entities are created from unresolved links, a relink
// binds them without touching any content.
func (h *APIHandler) relinkArticle(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	articleID := ps.ByName("id")

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})

	var article core.WikiArticle
	article.SetID(articleID)
	exists, err := orm.GetV2(ctx, &article)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, articleID)
		return
	}

	persistLinkedPages(&article)
	h.WriteGetOKJSON(w, article.ID, util.MapStr{"linked_pages": article.LinkedPages})
}
