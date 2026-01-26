/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"net/http"
	"time"

	"infini.sh/coco/core"

	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

const (
	AssistantTypeSimple           = "simple"
	AssistantTypeDeepThink        = "deep_think"
	AssistantTypeExternalWorkflow = "external_workflow"

	AssistantCachePrimary = "assistant"
)

// GetAssistant retrieves the assistant object from the cache or database.
func GetAssistant(req *http.Request, assistantID string) (*core.Assistant, bool, error) {
	ctx := orm.NewContextWithParent(req.Context())
	return InternalGetAssistant(ctx, assistantID)
}

func InternalGetAssistant(ctx *orm.Context, assistantID string) (*core.Assistant, bool, error) {
	item := GeneralObjectCache.Get(AssistantCachePrimary, assistantID)
	var assistant *core.Assistant
	if item != nil && !item.Expired() {
		var ok bool
		if assistant, ok = item.Value().(*core.Assistant); ok {
			return assistant, true, nil
		}
	}
	assistant = &core.Assistant{}
	assistant.ID = assistantID
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "assistant")

	exists, err := orm.GetV2(ctx, assistant)
	if err != nil {
		return nil, exists, err
	}

	//expand datasource is the datasource is `*`
	if util.ContainsAnyInArray("*", assistant.Datasource.IDs) {
		ids, err := GetAllEnabledDatasourceIDs()
		if err != nil {
			panic(err)
		}
		assistant.Datasource.SetIDs(ids)
	}

	if util.ContainsAnyInArray("*", assistant.MCPConfig.IDs) {
		ids, err := GetAllEnabledMCPServerIDs()
		if err != nil {
			panic(err)
		}
		assistant.MCPConfig.SetIDs(ids)
	}

	//set default value
	if assistant.MCPConfig.MaxIterations <= 1 {
		assistant.MCPConfig.MaxIterations = 5
	}

	if assistant.AnsweringModel.PromptConfig == nil {
		assistant.AnsweringModel.PromptConfig = &core.PromptConfig{PromptTemplate: GenerateAnswerPromptTemplate}
	} else if assistant.AnsweringModel.PromptConfig.PromptTemplate == "" {
		assistant.AnsweringModel.PromptConfig.PromptTemplate = GenerateAnswerPromptTemplate
	}

	if assistant.Type == AssistantTypeDeepThink {
		deepThinkCfg := core.DeepThinkConfig{}
		buf := util.MustToJSONBytes(assistant.Config)
		util.MustFromJSONBytes(buf, &deepThinkCfg)

		if deepThinkCfg.IntentAnalysisModel.PromptConfig == nil {
			deepThinkCfg.IntentAnalysisModel.PromptConfig = &core.PromptConfig{PromptTemplate: QueryIntentPromptTemplate}
		} else if deepThinkCfg.IntentAnalysisModel.PromptConfig.PromptTemplate == "" {
			deepThinkCfg.IntentAnalysisModel.PromptConfig.PromptTemplate = QueryIntentPromptTemplate
		}

		assistant.Config = deepThinkCfg
		assistant.DeepThinkConfig = &deepThinkCfg
	}

	if assistant.RolePrompt == "" {
		assistant.RolePrompt = "You are a personal AI assistant designed by Coco AI(https://coco.rs), the backend team is behind INFINI Labs(https://infinilabs.com)."
	}

	// Cache the assistant object
	GeneralObjectCache.Set(AssistantCachePrimary, assistantID, assistant, time.Duration(30)*time.Minute)
	return assistant, true, nil
}

var TotalAssistantsCacheKey = "total_assistants"

func ClearAssistantsCache() {
	GeneralObjectCache.Delete(AssistantCachePrimary, TotalAssistantsCacheKey)
}

func CountAssistants() (int64, error) {
	item := GeneralObjectCache.Get(AssistantCachePrimary, TotalAssistantsCacheKey)
	var assistantCache int64
	if item != nil && !item.Expired() {
		var ok bool
		if assistantCache, ok = item.Value().(int64); ok {
			return assistantCache, nil
		}
	}

	queryDsl := util.MapStr{
		"query": util.MapStr{
			"term": util.MapStr{
				"enabled": true,
			},
		},
	}
	count, err := orm.Count(core.Assistant{}, util.MustToJSONBytes(queryDsl))
	if err == nil {
		GeneralObjectCache.Set(AssistantCachePrimary, TotalAssistantsCacheKey, count, time.Duration(30)*time.Minute)
	}

	return count, err
}
