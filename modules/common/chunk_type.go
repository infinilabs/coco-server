/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

// Define constants for the various stages
const (
	QueryIntent  = "query_intent"
	Tools        = "tools"
	QueryRewrite = "query_rewrite"
	FetchSource  = "fetch_source"
	PickSource   = "pick_source"
	DeepRead     = "deep_read"
	Think        = "think"    //reasoning message by LLM
	Response     = "response" //formal response by assistant
	References   = "references"
	ReplyEnd     = "reply_end"
)
