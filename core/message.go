/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"infini.sh/framework/core/orm"
)

type MessageRequest struct {
	Message     string   `config:"message" json:"message,omitempty" elastic_mapping:"message:{type:text}"`
	Attachments []string `config:"attachments" json:"attachments,omitempty"`
}

func (m *MessageRequest) IsEmpty() bool {
	return m.Message == "" && len(m.Attachments) == 0
}

// ChatMessage is a persisted chat message stored in Elasticsearch.
// Each user question or assistant reply is saved as one ChatMessage record.
type ChatMessage struct {
	orm.ORMObjectBase
	MessageType string   `json:"type"` // user, assistant, system
	SessionID   string   `json:"session_id"`
	From        string   `json:"from"`
	To          string   `json:"to,omitempty"`
	Message     string   `config:"message" json:"message,omitempty" elastic_mapping:"message:{type:text}"`
	Attachments []string `config:"attachments" json:"attachments,omitempty"`

	ReplyMessageID string `config:"reply_to_message" json:"reply_to_message,omitempty" elastic_mapping:"reply_to_message:{type:keyword}"`

	// Details holds the ordered list of processing steps performed during an assistant reply.
	// Each entry has a Type that determines the schema of its Payload. Known types:
	//
	//   Type="query_intent"  (Order=10)  — Payload: QueryIntent object
	//     {"category":"", "intent":"", "query":[], "keyword":[], "suggestion":[],
	//      "need_plan_tasks":bool, "need_call_tools":bool, "need_network_search":bool}
	//
	//   Type="fetch_source"  (Order=20)  — Payload: array of candidate documents
	//     [{"id":"", "title":"", "updated":"", "category":"", "summary":"(≤500 chars)", "url":""}]
	//
	//   Type="pick_source"   (Order=30)  — Payload: array of LLM-selected documents
	//     [{"id":"", "title":"", "explain":""}]
	//
	//   Type="deep_read"     (Order=40)  — Payload: nil; Description contains "Analyzing: <title>\n" per doc
	//
	//   Type="think"         (Order=50)  — Payload: nil; Description contains the LLM reasoning trace text
	//
	//   Type="deep_research" (Order=10)  — Payload: array of ChunkRecord from deep research v2
	//     [{"chunk_type":"research_planner_start|…|research_reporter_end", "message_chunk":"JSON string"}]
	//     These are the serialized streaming chunks replayed by the frontend to reconstruct the research UI.
	Details []ProcessingDetails `json:"details"`

	UpVote      int    `json:"up_vote"`
	DownVote    int    `json:"down_vote"`
	AssistantID string `json:"assistant_id"`

	// Payload carries top-level structured data for the entire message.
	// Currently only set by deep_research_v2 when a research report is generated:
	//   {"title":"", "url":"", "created":"", "attachment":"<attachment_id>", "format":"md|html"}
	// For all other message types this field is nil.
	Payload interface{} `json:"payload"`
}

// ProcessingDetails is one step in the assistant's processing pipeline.
// Order controls display/sort priority (lower = earlier). Type identifies the step kind.
// Either Description or Payload (or both) carry the step's data — see ChatMessage.Details for
// the full Type→Payload schema.
type ProcessingDetails struct {
	Order       int         `json:"order"`
	Type        string      `json:"type"`        // One of: query_intent, fetch_source, pick_source, deep_read, think, deep_research
	Description string      `json:"description"` // Human-readable text (used by deep_read and think when Payload is nil)
	Payload     interface{} `json:"payload"`     // Structured data; concrete type depends on Type (see ChatMessage.Details doc)
}

// MessageChunk is a streaming fragment sent to the client over chunked HTTP.
// The LLM response and system status updates are delivered as a sequence of these chunks in real time.
type MessageChunk struct {
	SessionId      string `json:"session_id"`
	MessageId      string `json:"message_id"`
	MessageType    string `json:"message_type"`
	ReplyToMessage string `json:"reply_to_message"`
	ChunkSequence  int    `json:"chunk_sequence"`
	ChunkType      string `json:"chunk_type"`
	MessageChunk   string `json:"message_chunk"`
	Streaming      bool   `json:"streaming,omitempty"`
	ContentType    string `json:"content_type,omitempty"`
}

func NewMessageChunk(sessionId, messageId, messageType, replyToMessage, chunkType, messageChunk string, chunkSequence int) *MessageChunk {
	return &MessageChunk{
		SessionId:      sessionId,
		MessageId:      messageId,
		MessageType:    messageType,
		ReplyToMessage: replyToMessage,
		ChunkSequence:  chunkSequence,
		ChunkType:      chunkType,
		MessageChunk:   messageChunk,
	}
}
