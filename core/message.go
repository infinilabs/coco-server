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

type ChatMessage struct {
	orm.ORMObjectBase
	MessageType string   `json:"type"` // user, assistant, system
	SessionID   string   `json:"session_id"`
	From        string   `json:"from"`
	To          string   `json:"to,omitempty"`
	Message     string   `config:"message" json:"message,omitempty" elastic_mapping:"message:{type:text}"`
	Attachments []string `config:"attachments" json:"attachments,omitempty"`

	ReplyMessageID string              `config:"reply_to_message" json:"reply_to_message,omitempty" elastic_mapping:"reply_to_message:{type:keyword}"`
	Details        []ProcessingDetails `json:"details"`
	UpVote         int                 `json:"up_vote"`
	DownVote       int                 `json:"down_vote"`
	AssistantID    string              `json:"assistant_id"`

	Payload interface{} `json:"payload"`
}

type ProcessingDetails struct {
	Order       int         `json:"order"`
	Type        string      `json:"type"` //chunk_type
	Description string      `json:"description"`
	Payload     interface{} `json:"payload"` //<Payload>{JSON}</Payload>
}

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
