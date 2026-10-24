/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

// Skill categories for the built-in seeds; advisory only — any string is
// accepted so users can organize their own.
const (
	SkillCategoryRetrieval    = "retrieval"
	SkillCategoryKnowledge    = "knowledge"
	SkillCategoryOnboarding   = "onboarding"
	SkillCategoryGuardrails   = "guardrails"
	SkillCategoryProductivity = "productivity"
)

// A Skill is a Markdown instruction block that is injected into the
// assistant's system prompt when enabled. It shapes HOW the assistant
// approaches a class of problems (e.g. "act as a retrieval expert") without
// adding executable tools — the assistant reuses its existing tool set.
type Skill struct {
	CombinedFullText
	Name        string `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext}"` // stable slug, e.g. "retrieval-expert"; idempotency key for seeds
	Title       string `json:"title" elastic_mapping:"title:{type:text,copy_to:combined_fulltext,fields:{keyword: {type: keyword}}}"`
	Description string `json:"description,omitempty" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"`
	Category    string `json:"category,omitempty" elastic_mapping:"category:{type:keyword,copy_to:combined_fulltext}"`
	// the Markdown body injected into the system prompt; stored but never searched
	Instructions string `json:"instructions,omitempty" elastic_mapping:"instructions:{enabled:false}"`
	Enabled      bool   `json:"enabled" elastic_mapping:"enabled:{type:boolean}"`
	// built-in seeds: editable and disable-able, but not deletable
	Builtin   bool   `json:"builtin" elastic_mapping:"builtin:{type:boolean}"`
	Icon      string `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"`
	SortOrder int    `json:"sort_order,omitempty" elastic_mapping:"sort_order:{type:integer}"`
	UpdatedBy string `json:"updated_by,omitempty" elastic_mapping:"updated_by:{enabled:false}"`
}
