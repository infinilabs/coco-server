/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import "infini.sh/framework/core/orm"

// AssistantTemplate is a scenario preset (customer support, sales pre-sales,
// IT helpdesk, HR policy Q&A, ...): a curated role prompt plus suggested
// questions and default tool bindings. Instantiating copies everything into
// a real Assistant — templates themselves never answer chats.
//
// Templates are seeded idempotently by Name (modules/assistant/template_seeds.go);
// user edits to seeded records survive restarts.
type AssistantTemplate struct {
	orm.ORMObjectBase
	Name               string       `json:"name" elastic_mapping:"name:{type:keyword}"` // stable slug, seed idempotency key
	Title              string       `json:"title" elastic_mapping:"title:{type:keyword,copy_to:combined_fulltext}"`
	Category           string       `json:"category,omitempty" elastic_mapping:"category:{type:keyword}"` // support | sales | hr | it | productivity
	Icon               string       `json:"icon" elastic_mapping:"icon:{enabled:false}"`
	Description        string       `json:"description,omitempty" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"`
	RolePrompt         string       `json:"role_prompt,omitempty" elastic_mapping:"role_prompt:{enabled:false}"`
	SuggestedQuestions []string     `json:"suggested_questions,omitempty" elastic_mapping:"suggested_questions:{type:keyword}"`
	ToolsConfig        ToolsConfig  `json:"tools,omitempty" elastic_mapping:"tools:{type:object,enabled:false}"`
	MCPConfig          MCPConfig    `json:"mcp_servers,omitempty" elastic_mapping:"mcp_servers:{type:object,enabled:false}"`
	ChatSettings       ChatSettings `json:"chat_settings,omitempty" elastic_mapping:"chat_settings:{type:object,enabled:false}"`
	Builtin            bool         `json:"builtin" elastic_mapping:"builtin:{type:boolean}"`
	SortOrder          int          `json:"sort_order" elastic_mapping:"sort_order:{type:integer}"`
}
