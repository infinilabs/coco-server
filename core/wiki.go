/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"time"

	"infini.sh/framework/core/orm"
)

// Wiki knowledge-hub objects (WS10 design doc §3).
//
// Visibility/status enums are stored as plain strings; the wiki module
// validates transitions at the API layer.

const (
	WikiVisibilityPublic  = "public"
	WikiVisibilityPrivate = "private"
	WikiVisibilityTeam    = "team"

	WikiArticleDraft     = "draft"
	WikiArticleReviewed  = "reviewed"
	WikiArticlePublished = "published"
	WikiArticleArchived  = "archived"

	WikiChangeAIGenerated = "ai-generated"
	WikiChangeHumanEdited = "human-edited"
	WikiChangeAutoUpdated = "auto-updated"

	WikiPageTypeEntity  = "entity"
	WikiPageTypeConcept = "concept"
	WikiPageTypeSource  = "source"

	// entity lifecycle (design doc D1): extraction proposes, humans
	// review/publish — the AI pipeline never publishes directly
	WikiEntityProposed  = "proposed"
	WikiEntityReviewed  = "reviewed"
	WikiEntityPublished = "published"

	// relation written by the wikilink parser when an entity page links to
	// another entity without an explicit typed relation
	WikiRelationMentions = "mentions"

	// governance proposal lifecycle: the scanner only proposes, a human
	// resolves or dismisses (D1 applies to knowledge governance too)
	WikiGovernanceOpen      = "open"
	WikiGovernanceResolved  = "resolved"
	WikiGovernanceDismissed = "dismissed"

	// governance proposal types
	WikiGovernanceStale      = "stale"       // cited docs changed / pending auto-updated version
	WikiGovernanceDuplicate  = "duplicate"   // near-identical article in the same KB
	WikiGovernanceConflict   = "conflict"    // regenerated version contradicts live content
	WikiGovernanceLowQuality = "low_quality" // low confidence or zero citations while published
	WikiGovernanceOrphan     = "orphan"      // article missing from the KB TOC tree

	// entity-dimension proposals filed by the extraction pipeline: the same
	// human gate, different subject
	WikiGovernanceEntityDuplicate = "entity_duplicate" // same name/alias, looks like one entity
	WikiGovernanceEntityConflict  = "entity_conflict"  // same name/alias but conflicting types
)

// WikiWorkspace groups knowledge bases per team or tenant.
type WikiWorkspace struct {
	orm.ORMObjectBase
	Name        string `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext,fields:{text:{type:text},pinyin:{type:text,analyzer:pinyin_analyzer}}}"`
	Description string `json:"description,omitempty" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"`
}

// WikiKnowledgeBase is a curated wiki bound to a set of datasources and
// optionally an assistant that maintains it (design doc D3: visibility
// public/private/team, team goes through the framework share module).
type WikiKnowledgeBase struct {
	orm.ORMObjectBase
	Name          string   `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext,fields:{text:{type:text},pinyin:{type:text,analyzer:pinyin_analyzer}}}"`
	Description   string   `json:"description,omitempty" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"`
	Icon          string   `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"`
	Visibility    string   `json:"visibility" elastic_mapping:"visibility:{type:keyword}"` // public | private | team
	WorkspaceID   string   `json:"workspace_id,omitempty" elastic_mapping:"workspace_id:{type:keyword}"`
	DatasourceIDs []string `json:"datasource_ids,omitempty" elastic_mapping:"datasource_ids:{type:keyword}"`
	AssistantID   string   `json:"assistant_id,omitempty" elastic_mapping:"assistant_id:{type:keyword}"`
	SyncStrategy  string   `json:"sync_strategy,omitempty" elastic_mapping:"sync_strategy:{type:keyword}"` // realtime | scheduled | manual
	ArticleCount  int      `json:"article_count" elastic_mapping:"article_count:{type:integer}"`
	// runtime status of the KB agent pipeline, not a query target
	AIStatus string `json:"ai_status,omitempty" elastic_mapping:"ai_status:{enabled:false}"`
}

// WikiSourceReference points back at the ingested document a statement
// came from (design doc D2: doc_id + locator are the hard contract for
// citation backlinks, aligned with deep_research MaterialReference).
type WikiSourceReference struct {
	DocID      string `json:"doc_id" elastic_mapping:"doc_id:{type:keyword}"`
	SourceType string `json:"source_type,omitempty" elastic_mapping:"source_type:{type:keyword}"`
	SourceName string `json:"source_name,omitempty" elastic_mapping:"source_name:{type:keyword}"`
	Title      string `json:"title,omitempty" elastic_mapping:"title:{type:text}"`
	URL        string `json:"url,omitempty" elastic_mapping:"url:{enabled:false}"`
	Excerpt    string `json:"excerpt,omitempty" elastic_mapping:"excerpt:{type:text}"`
	Locator    string `json:"locator,omitempty" elastic_mapping:"locator:{type:keyword}"` // page/section/clause
}

// WikiArticle is a curated page in structured markdown (design doc §3.3:
// Obsidian-style sections with [[type:name]] wikilinks and doc_id mentions).
type WikiArticle struct {
	orm.ORMObjectBase
	KbID        string                `json:"kb_id" elastic_mapping:"kb_id:{type:keyword}"`
	TocNodeID   string                `json:"toc_node_id,omitempty" elastic_mapping:"toc_node_id:{type:keyword}"`
	Title       string                `json:"title" elastic_mapping:"title:{type:keyword,copy_to:combined_fulltext,fields:{text:{type:text},pinyin:{type:text,analyzer:pinyin_analyzer}}}"`
	Summary     string                `json:"summary,omitempty" elastic_mapping:"summary:{type:text,copy_to:combined_fulltext}"`
	Content     string                `json:"content,omitempty" elastic_mapping:"content:{enabled:false}"`    // full structured markdown, versions keep history
	PageType    string                `json:"page_type,omitempty" elastic_mapping:"page_type:{type:keyword}"` // entity | concept | source
	Subtype     string                `json:"subtype,omitempty" elastic_mapping:"subtype:{type:keyword}"`
	Aliases     []string              `json:"aliases,omitempty" elastic_mapping:"aliases:{type:keyword,copy_to:combined_fulltext}"`
	Tags        []string              `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword,copy_to:combined_fulltext}"`
	Status      string                `json:"status" elastic_mapping:"status:{type:keyword}"` // draft | reviewed | published | archived
	AIGenerated bool                  `json:"ai_generated" elastic_mapping:"ai_generated:{type:boolean}"`
	Confidence  string                `json:"confidence,omitempty" elastic_mapping:"confidence:{type:keyword}"` // high | medium | low
	Sources     []WikiSourceReference `json:"sources,omitempty" elastic_mapping:"sources:{type:object,enabled:false}"`
	EntityID    string                `json:"entity_id,omitempty" elastic_mapping:"entity_id:{type:keyword}"` // set when page_type=entity
	// wikilinks parsed from content at save time (design doc B3)
	LinkedPages []WikiLinkedPage `json:"linked_pages,omitempty" elastic_mapping:"linked_pages:{type:object,enabled:false}"`
}

// WikiTocNode is one node of a KB's table of contents tree.
type WikiTocNode struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Type      string `json:"type"` // folder | article
	ArticleID string `json:"article_id,omitempty"`
	Icon      string `json:"icon,omitempty"`
	// Children is self-referential; the tag walker cannot recurse it
	// (stack overflow) and the tree is stored under the parent's
	// nodes:{enabled:false} mapping anyway
	Children []WikiTocNode `json:"children,omitempty" elastic_mapping:"-"`
}

// WikiToc stores one document tree per KB (single object, updated wholesale).
type WikiToc struct {
	orm.ORMObjectBase
	KbID  string        `json:"kb_id" elastic_mapping:"kb_id:{type:keyword}"`
	Nodes []WikiTocNode `json:"nodes,omitempty" elastic_mapping:"nodes:{type:object,enabled:false}"`
}

// WikiVersion is an immutable full-content snapshot written by the API layer;
// change_type is injected server-side and never trusted from the client (D4).
type WikiVersion struct {
	orm.ORMObjectBase
	ArticleID     string `json:"article_id" elastic_mapping:"article_id:{type:keyword}"`
	Version       int    `json:"version" elastic_mapping:"version:{type:integer}"`
	ChangeType    string `json:"change_type" elastic_mapping:"change_type:{type:keyword}"` // ai-generated | human-edited | auto-updated
	ChangeSummary string `json:"change_summary,omitempty" elastic_mapping:"change_summary:{type:text}"`
	Content       string `json:"content,omitempty" elastic_mapping:"content:{enabled:false}"`
	CreatedBy     string `json:"created_by,omitempty" elastic_mapping:"created_by:{type:keyword}"`
}

// WikiBookmark pins an article for a user; owner_id isolates records.
type WikiBookmark struct {
	orm.ORMObjectBase
	ArticleID string `json:"article_id" elastic_mapping:"article_id:{type:keyword}"`
}

// WikiComment is a user comment on an article (discussion stays in the wiki;
// only article *content* goes through the review status machine).
type WikiComment struct {
	orm.ORMObjectBase
	ArticleID string `json:"article_id" elastic_mapping:"article_id:{type:keyword}"`
	UserID    string `json:"user_id" elastic_mapping:"user_id:{type:keyword}"`
	UserName  string `json:"user_name" elastic_mapping:"user_name:{type:keyword}"`
	Content   string `json:"content" elastic_mapping:"content:{type:text,copy_to:combined_fulltext}"`
}

// WikiNotification records wiki events targeted at a user.
type WikiNotification struct {
	orm.ORMObjectBase
	UserID     string `json:"user_id" elastic_mapping:"user_id:{type:keyword}"`
	TargetType string `json:"target_type" elastic_mapping:"target_type:{type:keyword}"` // article | kb | version
	TargetID   string `json:"target_id" elastic_mapping:"target_id:{type:keyword}"`
	Action     string `json:"action,omitempty" elastic_mapping:"action:{type:keyword}"` // ai-draft | status-change | ...
	Message    string `json:"message,omitempty" elastic_mapping:"message:{type:text}"`
	Read       bool   `json:"read" elastic_mapping:"read:{type:boolean}"`
}

// WikiGovernanceProposal is one AI-spotted knowledge-governance item: the
// background scanner detects stale / duplicate / conflicting / low-quality /
// orphaned articles and files a proposal; applying any fix stays a human
// decision on the governance queue (D1).
type WikiGovernanceProposal struct {
	orm.ORMObjectBase
	KbID         string                 `json:"kb_id" elastic_mapping:"kb_id:{type:keyword}"`
	ArticleID    string                 `json:"article_id" elastic_mapping:"article_id:{type:keyword}"`
	ArticleTitle string                 `json:"article_title,omitempty" elastic_mapping:"article_title:{type:keyword}"`
	Type         string                 `json:"type" elastic_mapping:"type:{type:keyword}"` // stale | duplicate | conflict | low_quality | orphan
	Status       string                 `json:"status" elastic_mapping:"status:{type:keyword}"`
	Reason       string                 `json:"reason,omitempty" elastic_mapping:"reason:{type:text}"`
	Evidence     map[string]interface{} `json:"evidence,omitempty" elastic_mapping:"evidence:{type:object,enabled:false}"`
	ResolvedBy   string                 `json:"resolved_by,omitempty" elastic_mapping:"resolved_by:{type:keyword}"`
	ResolvedAt   *time.Time             `json:"resolved_at,omitempty" elastic_mapping:"resolved_at:{type:date}"`
}

// WikiLike is a user's thumbs-up on an article; one row per (article, user),
// toggled by create/delete only.
type WikiLike struct {
	orm.ORMObjectBase
	ArticleID string `json:"article_id" elastic_mapping:"article_id:{type:keyword}"`
	UserID    string `json:"user_id" elastic_mapping:"user_id:{type:keyword}"`
	UserName  string `json:"user_name,omitempty" elastic_mapping:"user_name:{type:keyword}"`
}

// WikiEntityRelation is a typed edge to another entity.
type WikiEntityRelation struct {
	TargetID   string `json:"target_id"`
	TargetType string `json:"target_type,omitempty"`
	Relation   string `json:"relation"`             // belongs_to | promotes | leases | mentions | ...
	Provenance string `json:"provenance,omitempty"` // source doc_id or article_id
}

// WikiLinkedPage is a wikilink [[type:name]] parsed out of article content
// at save time (design doc B3): stored redundantly so rendering and graph
// walks don't re-parse full content.
type WikiLinkedPage struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	EntityID string `json:"entity_id,omitempty"` // resolved against coco entities, empty when unknown
}

// WikiEntity is the ontology query face; the curated page (WikiArticle,
// page_type=entity) links back via EntityID/ArticleID (design doc D1).
type WikiEntity struct {
	orm.ORMObjectBase
	Type       string                 `json:"type" elastic_mapping:"type:{type:keyword}"` // store | campaign | contract | ...
	Subtype    string                 `json:"subtype,omitempty" elastic_mapping:"subtype:{type:keyword}"`
	Name       string                 `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext,fields:{text:{type:text},pinyin:{type:text,analyzer:pinyin_analyzer}}}"`
	Aliases    []string               `json:"aliases,omitempty" elastic_mapping:"aliases:{type:keyword,copy_to:combined_fulltext}"`
	Properties map[string]interface{} `json:"properties,omitempty" elastic_mapping:"properties:{type:object}"` // free attributes per ontology schema
	Relations  []WikiEntityRelation   `json:"relations,omitempty" elastic_mapping:"relations:{type:object,enabled:false}"`
	Status     string                 `json:"status" elastic_mapping:"status:{type:keyword}"` // proposed | reviewed | published
	Confidence float64                `json:"confidence,omitempty" elastic_mapping:"confidence:{type:float}"`
	ArticleID  string                 `json:"article_id,omitempty" elastic_mapping:"article_id:{type:keyword}"`
	// documents this entity was extracted from (bidirectional with
	// Document.EntityIDs, design doc B3)
	Sources []WikiSourceReference `json:"sources,omitempty" elastic_mapping:"sources:{type:object,enabled:false}"`
}

// WikiOntologySchema defines the entity/relation vocabulary for a KB or
// tenant; maintained as configuration (design doc §3.5, no visual editor
// in the POC).
type WikiOntologySchema struct {
	orm.ORMObjectBase
	Scope  string                 `json:"scope" elastic_mapping:"scope:{type:keyword}"` // kb:<id> | tenant
	Schema map[string]interface{} `json:"schema" elastic_mapping:"schema:{type:object,enabled:false}"`
}
