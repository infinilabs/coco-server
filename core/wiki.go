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
	WikiPageTypeTopic   = "topic" // outline-driven subject page: a cluster's curated hub
	WikiPageTypeMap     = "map"   // hand-maintained company map: the agent entry page (D6)
	WikiPageTypePattern = "pattern" // maintainer-distilled failure pattern: 问题+根因+解法 (LLM-wiki P1)
	WikiPageTypeSkill   = "skill"   // proposer-distilled procedural skill: 适用条件+步骤+反例 (LLM-wiki P2)

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

	// Work-side signals (D5/D7): not tied to one article — ArticleID carries
	// the query hash (knowledge_gap) or the chat message id / content hash
	// (correction) so the scanner-style idempotency key still dedups.
	WikiGovernanceKnowledgeGap = "knowledge_gap" // query keeps returning zero hits
	WikiGovernanceCorrection   = "correction"    // user corrected an answer from work
	// schema drift (W10 close-out): an entity's type is no longer declared
	// in the ontology vocabulary — deterministic, no LLM confirmation needed
	WikiGovernanceDrift = "drift"

	// entity edit propagation (W10): a rename/alias/type/relations change
	// on an entity invalidates the articles that reference it — refresh
	// proposals keep the human gate (edits merge into the live article,
	// never automatically)
	WikiGovernanceArticleRefresh = "article_refresh"

	// entity-dimension proposals filed by the extraction pipeline: the same
	// human gate, different subject
	WikiGovernanceEntityDuplicate = "entity_duplicate" // same name/alias, looks like one entity
	WikiGovernanceEntityConflict  = "entity_conflict"  // same name/alias but conflicting types

	// the compiler's suggested TOC organization (dimension folders: 客户与组织,
	// 产品与服务, 专题与概念). Confirming the proposal in the governance queue
	// applies the tree; dismissing drops it.
	WikiGovernanceTreeOrganize = "tree_organize"

	// the compiler's page proposals: REDUCE'd statements awaiting a draft
	// page (apply generates it from the evidence)
	WikiGovernanceKnowledgeCompile = "knowledge_compile"

	// maintainer-distilled pattern proposals (LLM-wiki P1): the distiller
	// mines experience records and files page creates/patches — same human
	// gate as every other proposal (auto-approve KBs apply them like
	// compile drafts)
	WikiGovernancePatternCreate = "pattern_create"
	WikiGovernancePatternUpdate = "pattern_update"

	// proposer-distilled skill proposals (LLM-wiki P2): one atomic skill
	// create/patch per proposer run, gated by the reviewer LLM before any
	// apply
	WikiGovernanceSkillCreate = "skill_create"
	WikiGovernanceSkillPatch  = "skill_patch"

	// impact-driven retention (LLM-wiki P3): a long-idle, never-cited page
	// becomes an archive candidate so active knowledge gets the room
	WikiGovernanceImpactArchive = "impact_archive"

	// the EINO compile agent's output (DESIGN-compile-agent.md): a full
	// replacement body for one topic page, grounded in tool-fetched
	// evidence, filed as a proposal so the usual gate applies
	WikiGovernanceCompilePatch = "compile_patch"
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
	// 企业目录布局(可选):研发/销售/售前/市场等部门各成顶层目录,目录内
	// 编译器按实体类型维度再分。部门归属按被引文档的 folder_path 前缀匹配;
	// 未配置时编译器退回单层维度目录
	TocLayout []WikiTocLayoutFolder `json:"toc_layout,omitempty" elastic_mapping:"toc_layout:{type:object}"`
	// runtime status of the KB agent pipeline, not a query target
	AIStatus string `json:"ai_status,omitempty" elastic_mapping:"ai_status:{enabled:false}"`
	// AutoApprove (自动审批): entities extracted for this KB publish
	// directly and governance proposals auto-apply — the pipeline updates
	// the KB without a human gate. Opt-in per KB; corrections and
	// knowledge-gap signals stay manual regardless.
	// no omitempty on these bools: a false would serialize away and the
	// partial-update merge would keep a stored true forever — flags must be
	// clearable, not just settable
	AutoApprove bool `json:"auto_approve" elastic_mapping:"auto_approve:{type:boolean}"`
	// ExperienceCapture (经验捕获, LLM-wiki P1): opt-in per KB. When on,
	// work signals on answers from the KB's bound assistant (corrections,
	// votes) land as WikiExperience records the distiller mines. Off by
	// default — traces contain user questions.
	ExperienceCapture bool `json:"experience_capture" elastic_mapping:"experience_capture:{type:boolean}"`
}

// WikiTocLayoutFolder is one top-level department folder of the KB layout.
// PathPrefixes match the cited documents' folder path (any-level prefix,
// case-insensitive) — a doc under "kb/研发/" files its articles under the
// department whose prefix matches first. Dimensions renames the per-department
// dimension folders in canonical slot order (organization/product/topic);
// unset or empty slots fall back to the shared defaults.
type WikiTocLayoutFolder struct {
	Title        string   `json:"title"`                   // 研发部
	PathPrefixes []string `json:"path_prefixes,omitempty"` // ["研发", "rd"]
	Dimensions   []string `json:"dimensions,omitempty"`    // ["技术方案", "产品与服务", "架构决策"]
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

// WikiSection is one planned section of a topic page (outline-driven
// compilation): the heading the planner emitted, its drafted body and the
// documents backing it. Enrichment merges at THIS granularity — a new
// compile updates the sections it touches instead of stacking a new page.
type WikiSection struct {
	ID        string   `json:"id"`
	Heading   string   `json:"heading"`
	Body      string   `json:"body,omitempty"`
	DocIDs    []string `json:"doc_ids,omitempty"`
	UpdatedAt int64    `json:"updated_at,omitempty"`
}

// WikiPageMaturity records how complete a compiled topic page is: distinct
// cited documents, filled vs planned sections, and a 0-1 coverage score.
// The TOC badge and the consolidate (thin-page absorb) rule both read it.
type WikiPageMaturity struct {
	Sources  int     `json:"sources,omitempty"`
	Sections int     `json:"sections,omitempty"`
	Planned  int     `json:"planned,omitempty"`
	Score    float64 `json:"score,omitempty"`
}

// Lint warning codes (computed on save, advisory only — lint never blocks
// a write, it tells the author and the compiler what to fix).
const (
	WikiLintDeadLink          = "dead_link"          // [[name]] resolves to no entity or article in this KB
	WikiLintThinContent       = "thin_content"       // body under the readability floor
	WikiLintNoSources         = "no_sources"         // concept/topic page with zero cited documents
	WikiLintMissingProvenance = "missing_provenance" // skill page with no pattern provenance (LLM-wiki P2)
)

// WikiLintWarning is one advisory schema/consistency finding computed at
// save time (wiki-skills doctrine: lint per page, audit per KB). Stored on
// the article so readers and the editor banner see it without re-running
// resolution.
type WikiLintWarning struct {
	Code    string `json:"code" elastic_mapping:"code:{type:keyword}"`
	Field   string `json:"field,omitempty" elastic_mapping:"field:{type:keyword}"`
	Message string `json:"message" elastic_mapping:"message:{type:text}"`
	Detail  string `json:"detail,omitempty" elastic_mapping:"detail:{type:keyword}"`
}

// WikiImpact is the usage-side counter of a page (the skill-impact.md
// idea): how often the answer pipeline actually pulled the page into a
// model context. Written programmatically by the flush task — never by
// the LLM, never by hand.
type WikiImpact struct {
	AnswerHits int        `json:"answer_hits,omitempty" elastic_mapping:"answer_hits:{type:integer}"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty" elastic_mapping:"last_used_at:{type:date}"`
}

// Evolution-log kinds — the KB-level decision record (the evolution log of
// WikiSkill): what changed, who decided, and why. Append-only; entries are
// never rewritten or rolled back, rejections included, so downstream
// proposers can see what was already tried.
const (
	WikiEvolutionProposalApplied   = "proposal_applied"   // a governance proposal's fix executed
	WikiEvolutionProposalDismissed = "proposal_dismissed" // a human retired a proposal without a fix
	WikiEvolutionPatternDistilled  = "pattern_distilled"  // a distill run sampled experiences and filed pattern proposals
	WikiEvolutionSkillProposed     = "skill_proposed"     // a proposer run (accepted file, rejection or no_action)
	WikiEvolutionGateVerdict       = "gate_verdict"       // the reviewer LLM judged a skill proposal
	WikiEvolutionCompileAgent      = "compile_agent"      // one EINO compile-agent iteration (patch filed or no_action)
)

// WikiEvolutionEntry is one append-only line of a KB's decision log.
// Written by the programs that apply/dismiss governance proposals (the
// auto-approve sweep writes actor="auto-approve"); P1 evolution runs will
// append their own kinds on top.
type WikiEvolutionEntry struct {
	orm.ORMObjectBase
	KbID       string                 `json:"kb_id" elastic_mapping:"kb_id:{type:keyword}"`
	Kind       string                 `json:"kind" elastic_mapping:"kind:{type:keyword}"`
	Actor      string                 `json:"actor,omitempty" elastic_mapping:"actor:{type:keyword}"` // user id | auto-approve
	TargetType string                 `json:"target_type,omitempty" elastic_mapping:"target_type:{type:keyword}"`
	TargetID   string                 `json:"target_id,omitempty" elastic_mapping:"target_id:{type:keyword}"`
	Summary    string                 `json:"summary,omitempty" elastic_mapping:"summary:{type:text}"`
	Detail     map[string]interface{} `json:"detail,omitempty" elastic_mapping:"detail:{type:object,enabled:false}"`
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
	// page-level attachments (core.Attachment ids): files hung on the wiki
	// page itself, N per page — a page-level concern, independent of the
	// cited source documents (a source document IS the file)
	Attachments []string `json:"attachments,omitempty" elastic_mapping:"attachments:{type:keyword}"`
	// topic-page structure (outline-driven compilation): planned sections
	// with per-section provenance. Content stays the rendered markdown of
	// the same sections — search, wikilinks and versions keep working off
	// Content, Outline is the structured source of truth for enrichment.
	Outline     []WikiSection     `json:"outline,omitempty" elastic_mapping:"outline:{type:object,enabled:false}"`
	KeyEntities []string          `json:"key_entities,omitempty" elastic_mapping:"key_entities:{type:keyword}"`
	Maturity    *WikiPageMaturity `json:"maturity,omitempty" elastic_mapping:"maturity:{type:object,enabled:false}"`
	// advisory findings recomputed on every save (dead links, thin body,
	// missing citations) — lint informs, it never blocks
	Lint []WikiLintWarning `json:"lint,omitempty" elastic_mapping:"lint:{type:object,enabled:false}"`
	// usage telemetry maintained by the impact flush task, not by writers
	Impact *WikiImpact `json:"impact,omitempty" elastic_mapping:"impact:{type:object,enabled:false}"`
	// structured pattern payload of a page_type=pattern page (LLM-wiki P1):
	// the maintainer's root-cause analysis in queryable form; Content keeps
	// the rendered markdown
	Pattern *WikiPattern `json:"pattern,omitempty" elastic_mapping:"pattern:{type:object,enabled:false}"`
	// structured skill payload of a page_type=skill page (LLM-wiki P2):
	// procedural knowledge the assistants load on demand
	Skill *WikiSkill `json:"skill,omitempty" elastic_mapping:"skill:{type:object,enabled:false}"`
}

// WikiPattern is the structured body of a pattern page (the WikiSkill
// pattern-page shape: PROBLEM + ROOT CAUSE + FIX in 10-30 lines). Evidence
// points at the experience records / documents it was distilled from.
type WikiPattern struct {
	Symptom   string                `json:"symptom,omitempty" elastic_mapping:"symptom:{type:text}"`
	RootCause string                `json:"root_cause,omitempty" elastic_mapping:"root_cause:{type:text}"`
	Fix       string                `json:"fix,omitempty" elastic_mapping:"fix:{type:text}"`
	Evidence  []WikiSourceReference `json:"evidence,omitempty" elastic_mapping:"evidence:{type:object,enabled:false}"`
}

// SkillLoad is one conditionally-loaded attachment of a skill page (the
// graph-layer `loads` field): a file reference and when to pull it in.
type SkillLoad struct {
	Path string `json:"path" elastic_mapping:"path:{type:keyword}"`
	When string `json:"when,omitempty" elastic_mapping:"when:{type:text}"`
}

// SkillReachFor is when a skill "reaches" a task at all (the graph-layer
// `reach-for` field): the tier-1 index carries these so the assistant can
// load the skill without reading it first.
type SkillReachFor struct {
	Phrases []string `json:"phrases,omitempty" elastic_mapping:"phrases:{type:keyword}"`
	Globs   []string `json:"globs,omitempty" elastic_mapping:"globs:{type:keyword}"`
}

// WikiSkill is the structured body of a skill page: executable procedural
// knowledge (the WikiSkill skills/ layer). Provenance (which pattern pages
// motivated this skill) lives in Sources as pattern-page references and in
// the content's [[pattern:...]] links — the PURPOSE.md semantics.
type WikiSkill struct {
	Applicability   string        `json:"applicability,omitempty" elastic_mapping:"applicability:{type:text}"`
	Steps           []string      `json:"steps,omitempty" elastic_mapping:"steps:{type:text}"`
	AntiPatterns    []string      `json:"anti_patterns,omitempty" elastic_mapping:"anti_patterns:{type:text}"`
	WhenNotToApply  []string      `json:"when_not_to_apply,omitempty" elastic_mapping:"when_not_to_apply:{type:text}"`
	ReachFor        *SkillReachFor `json:"reach_for,omitempty" elastic_mapping:"reach_for:{type:object,enabled:false}"`
	Loads           []SkillLoad   `json:"loads,omitempty" elastic_mapping:"loads:{type:object,enabled:false}"`
	// Rationale records why the proposer believed this skill earns its
	// place — surfaced in the gate review and the version history.
	Rationale string `json:"rationale,omitempty" elastic_mapping:"rationale:{type:text}"`
}

// experience outcome/feedback vocabularies
const (
	ExperienceFeedbackNegative = "negative"
	ExperienceFeedbackPositive = "positive"
	ExperienceFeedbackNone     = "none"

	ExperienceOutcomeResolved  = "resolved"
	ExperienceOutcomeAbandoned = "abandoned"
	ExperienceOutcomeEscalated = "escalated" // e.g. filed a correction
)

// WikiExperience is one captured work trace on the raw layer (LLM-wiki P1,
// the WikiSkill raw/ analog): what was asked, what the assistant answered,
// which documents it leaned on and how the human judged it. Records are
// immutable learning material — written once by capture hooks (corrections
// today, votes/API callers as they land), read by the distiller, expired by
// retention. KB-scoped and opt-in.
type WikiExperience struct {
	orm.ORMObjectBase
	KbID      string  `json:"kb_id" elastic_mapping:"kb_id:{type:keyword}"`
	Kind      string  `json:"kind,omitempty" elastic_mapping:"kind:{type:keyword}"` // chat | correction | gap
	SessionID string  `json:"session_id,omitempty" elastic_mapping:"session_id:{type:keyword}"`
	MessageID string  `json:"message_id,omitempty" elastic_mapping:"message_id:{type:keyword}"`
	Question  string  `json:"question,omitempty" elastic_mapping:"question:{type:text}"`
	Answer    string  `json:"answer,omitempty" elastic_mapping:"answer:{type:text}"`
	Citations []WikiSourceReference `json:"citations,omitempty" elastic_mapping:"citations:{type:object,enabled:false}"`
	Feedback  string  `json:"feedback,omitempty" elastic_mapping:"feedback:{type:keyword}"`
	Note      string  `json:"note,omitempty" elastic_mapping:"note:{type:text}"` // human comment / correction text
	RouteHint string  `json:"route_hint,omitempty" elastic_mapping:"route_hint:{type:keyword}"`
	Outcome   string  `json:"outcome,omitempty" elastic_mapping:"outcome:{type:keyword}"`
	// Retention is the expiry time; the distiller's sweep deletes expired
	// rows. Immutable-but-expiring: the raw layer never gets rewritten,
	// only dropped.
	Retention *time.Time `json:"retention,omitempty" elastic_mapping:"retention:{type:date}"`
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

// WikiArticleAttachment is the resolved metadata of one file hung on a wiki
// page. Attachments are a page-level concern: one article carries N files,
// independent of the cited source documents — a source document IS the file,
// it has no attachment list of its own in this model.
type WikiArticleAttachment struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Size     int64  `json:"size,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Created  string `json:"created,omitempty"`
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
	// KbID scopes the entity to one knowledge base: each KB compiles its
	// own entity pool from its bound datasources, and deleting the KB
	// deletes its entities wholesale. Same-name entities in different KBs
	// are different rows (isolated pools, per the per-KB review model).
	KbID       string                 `json:"kb_id,omitempty" elastic_mapping:"kb_id:{type:keyword}"`
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

// WikiCompileRules is the "knowledge compile rules" layer of the ontology
// schema (D6): how the curated layer is compiled and how its disputes are
// decided. It lives in core so both the wiki module (editor, governance)
// and the document module (the retrieval noise gate) read the same rules
// without an import cycle.
type WikiCompileRules struct {
	// ConceptMinSources is how many independent sources a concept page
	// needs before it enters retrieval (a single-source concept is a
	// hypothesis, not knowledge). Zero means the default of 2.
	ConceptMinSources int `json:"concept_min_sources,omitempty"`
	// ConflictStrategy says what the compiler does with contradictory
	// claims: mark (file a proposal, both sides stay visible) or isolate
	// (keep the lower-priority side out of retrieval). Default mark.
	ConflictStrategy string `json:"conflict_strategy,omitempty"`
	// SourcePriority orders sources (datasource ids or category keys,
	// highest first); conflicts resolve toward the higher-priority side.
	SourcePriority []string `json:"source_priority,omitempty"`
}

const (
	WikiConflictStrategyMark    = "mark"
	WikiConflictStrategyIsolate = "isolate"

	// DefaultConceptMinSources is the concept-page source floor when the
	// rules don't override it.
	DefaultConceptMinSources = 2
)

// Normalized clamps the rules to sane values and fills defaults.
func (r *WikiCompileRules) Normalized() *WikiCompileRules {
	if r == nil {
		r = &WikiCompileRules{}
	}
	if r.ConceptMinSources < 1 || r.ConceptMinSources > 10 {
		r.ConceptMinSources = DefaultConceptMinSources
	}
	if r.ConflictStrategy != WikiConflictStrategyIsolate {
		r.ConflictStrategy = WikiConflictStrategyMark
	}
	return r
}

// SourcePriorityRank returns the priority position of a source key (lower
// rank = higher priority); keys absent from the list rank lowest.
func (r *WikiCompileRules) SourcePriorityRank(source string) int {
	if r == nil {
		return 1 << 30
	}
	for i, key := range r.SourcePriority {
		if key == source {
			return i
		}
	}
	return 1 << 30
}

// WikiOntologySchema defines the entity/relation vocabulary for a KB or
// tenant; maintained as configuration (design doc §3.5, no visual editor
// in the POC).
type WikiOntologySchema struct {
	orm.ORMObjectBase
	Scope  string                 `json:"scope" elastic_mapping:"scope:{type:keyword}"` // kb:<id> | tenant
	Schema map[string]interface{} `json:"schema" elastic_mapping:"schema:{type:object,enabled:false}"`
}
