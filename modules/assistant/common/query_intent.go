package common

type QueryIntent struct {
	Category   string   `json:"category"`
	Intent     string   `json:"intent"`
	Query      []string `json:"query"`
	Keyword    []string `json:"keyword"`
	Suggestion []string `json:"suggestion"`

	NeedPlanTasks     bool `json:"need_plan_tasks"`     //if it is not a simple task
	NeedCallTools     bool `json:"need_call_tools"`     //if it is necessary
	NeedNetworkSearch bool `json:"need_network_search"` //if need external data sources
}
