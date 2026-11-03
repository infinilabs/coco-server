/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package core

import "infini.sh/framework/core/orm"

// SystemPromptNote joins the assistant system prompt so the model keeps
// memory promises honest: it cannot persist anything mid-conversation —
// distillation runs system-side after each exchange and candidates only
// take effect after the owner confirms them in the console.
const SystemPromptNote = "\n\nLong-term memory policy: you cannot save memories yourself. The system automatically distills candidate memories from each conversation exchange; they only take effect after the user confirms them in the console. Never claim you have \"saved\", \"stored\" or \"will permanently remember\" anything. When the user shares a durable preference or fact, acknowledge it naturally and, if useful, mention it will be remembered once confirmed."

// MemoryRecord (W6, confirmation-gated long-term memory): one durable
// fact about a user, distilled from a conversation or filed manually.
// Every record starts PENDING — nothing enters a prompt until the user
// confirms it (the D1 red line applied to memory). A new record about the
// same topic supersedes the old one only after confirmation; the old row
// stays as history.
type MemoryRecord struct {
	orm.ORMObjectBase

	// Kind is one of MemoryKind* — profile / preference / fact / task /
	// interest. profile+preference are resident context (length-capped),
	// fact+task are query-recalled, interest is bookkeeping.
	Kind string `json:"kind" elastic_mapping:"kind:{type:keyword}"`

	// Status: pending → confirmed | rejected; confirmed state records
	// (profile/preference) may later become superseded when a newer
	// record about the same topic replaces them. Only confirmed memories
	// inject anywhere or surface through search_memory; superseded rows
	// stay as history for trend/change analysis.
	Status string `json:"status" elastic_mapping:"status:{type:keyword}"`

	// UserID scopes every read and write — memories are strictly
	// per-user, no cross-user reads, admins included.
	UserID string `json:"user_id" elastic_mapping:"user_id:{type:keyword}"`

	// Content is the memory sentence itself, stored in the user's
	// language, phrased standalone ("Prefers replies in English").
	Content string `json:"content" elastic_mapping:"content:{type:text}"`

	// SupersedeOf points at the record this one replaces (same topic,
	// newer truth); the superseded row keeps its status for history.
	SupersedeOf string `json:"supersede_of,omitempty" elastic_mapping:"supersede_of:{type:keyword}"`

	// Context carries where the memory came from: session id, the
	// triggering exchange excerpt, distillation model. Review evidence,
	// never injected into prompts.
	Context map[string]interface{} `json:"context,omitempty" elastic_mapping:"context:{type:object,enabled:false}"`
}

const (
	MemoryKindProfile    = "profile"
	MemoryKindPreference = "preference"
	MemoryKindFact       = "fact"
	MemoryKindTask       = "task"
	MemoryKindInterest   = "interest"

	MemoryStatusPending    = "pending"
	MemoryStatusConfirmed  = "confirmed"
	MemoryStatusRejected   = "rejected"
	MemoryStatusSuperseded = "superseded"
)

// ValidMemoryKind reports whether k is one of the five kinds.
func ValidMemoryKind(k string) bool {
	switch k {
	case MemoryKindProfile, MemoryKindPreference, MemoryKindFact, MemoryKindTask, MemoryKindInterest:
		return true
	}
	return false
}

// ResidentMemoryKinds are injected into every turn (length-capped);
// the rest are recalled per query.
var ResidentMemoryKinds = []string{MemoryKindProfile, MemoryKindPreference}

// RecalledMemoryKinds are matched against the current query.
var RecalledMemoryKinds = []string{MemoryKindFact, MemoryKindTask}
