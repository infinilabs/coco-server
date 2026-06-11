/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

// Define constants for assistant streaming chunk types and related detail types.
const (
	// ReplyStart is a streaming chunk type that marks the beginning of an
	// assistant loop. It is not persisted into ChatMessage.Details.
	ReplyStart = "reply_start"

	// Response is a streaming chunk type for the assistant's formal answer text.
	// The accumulated answer is persisted on ChatMessage.Message.
	Response = "response"

	// ReplyEnd is a streaming chunk type that marks the terminal chunk of an
	// assistant loop. Its message_chunk is a JSON string:
	// {"reason":"completed|user_cancelled|error|timeout"}. When reason is
	// "error", it also includes {"error":"..."}. It is also persisted as a
	// ProcessingDetails entry with Type=reply_end and the same structured payload.
	ReplyEnd = "reply_end"

	// ReplyEnd loop exit reasons carried in the reply_end payload.
	// These values are not streaming chunk types.
	ReplyEndReasonCompleted     = "completed"
	ReplyEndReasonUserCancelled = "user_cancelled"
	ReplyEndReasonError         = "error"
	ReplyEndReasonTimeout       = "timeout"

	// AttachmentWaiting is a streaming chunk type emitted as a heartbeat while
	// waiting for attachment processing to finish. It is not persisted into
	// ChatMessage.Details.
	AttachmentWaiting = "attachment_waiting"

	// QueryIntent is a streaming chunk type for query analysis output. The parsed
	// QueryIntent object is persisted as a ProcessingDetails entry.
	QueryIntent = "query_intent"

	// Tools is a streaming chunk type for tool-call records. Each message_chunk
	// contains only the tool name, arguments, and output; the accumulated text is
	// persisted as a ProcessingDetails entry whose Description carries the same
	// content.
	Tools = "tools"

	// FetchSource is a streaming chunk type for fetched source previews. Candidate
	// documents are also persisted as a ProcessingDetails entry.
	FetchSource = "fetch_source"

	// PickSource is a streaming chunk type for LLM source-picking output. Picked
	// documents are also persisted as a ProcessingDetails entry.
	PickSource = "pick_source"

	// DeepRead is a streaming chunk type for document-reading progress. It is
	// persisted as a ProcessingDetails entry whose Description carries the text.
	DeepRead = "deep_read"

	// Think is a streaming chunk type for LLM reasoning output. It is persisted as
	// a ProcessingDetails entry whose Description carries the reasoning trace.
	Think = "think"

	// DeepResearch is a ProcessingDetails.Type used when persisted deep research
	// chunks are grouped into one detail entry; it is not a streaming chunk type.
	DeepResearch = "deep_research"

	// The research_* constants below are streaming chunk types emitted by deep
	// research v2. They are collected and persisted together inside one
	// ProcessingDetails entry with Type=deep_research, rather than as separate
	// ProcessingDetails entries.
	ResearchPlannerStart        = "research_planner_start"
	ResearchPlannerEnd          = "research_planner_end"
	ResearchPlanList            = "research_plan_list"
	ResearchPlanUpdated         = "research_plan_updated"
	ResearchResearcherStart     = "research_researcher_start"
	ResearchResearcherStepStart = "research_researcher_step_start"
	ResearchResearcherStepEnd   = "research_researcher_step_end"
	ResearchResearcherEnd       = "research_researcher_end"
	ResearchReporterStart       = "research_reporter_start"
	ResearchReporterEnd         = "research_reporter_end"
)
