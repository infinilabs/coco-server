/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

// Define constants for the various stages
const (
	//common
	ReplyStart = "reply_start"
	Response   = "response" //formal response by assistant
	ReplyEnd   = "reply_end"

	//deep think related
	QueryIntent  = "query_intent"
	Tools        = "tools"
	QueryRewrite = "query_rewrite"
	FetchSource  = "fetch_source"
	PickSource   = "pick_source"
	DeepRead     = "deep_read"
	Think        = "think" //reasoning message by LLM
	References   = "references"

	//deep research
	ResearchPlannerStart = "research_planner_start"
	ResearchPlannerEnd   = "research_planner_end"

	ResearchPlanList    = "research_plan_list"
	ResearchPlanUpdated = "research_plan_updated"

	ResearchResearcherStart = "research_researcher_start"

	ResearchResearcherStepStart = "research_researcher_step_start"
	ResearchResearcherStepEnd   = "research_researcher_step_end"

	ResearchResearcherEnd = "research_researcher_end"

	ResearchReporterStart = "research_reporter_start"
	ResearchReporterEnd   = "research_reporter_end"
)
