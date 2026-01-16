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
	Think        = "think"    //reasoning message by LLM
	References   = "references"

)
