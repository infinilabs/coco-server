/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package service

import (
	"context"
	"net/http"
	"time"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// GetAssistant retrieves the assistant object from the cache or database.
func GetAssistant(req *http.Request, assistantID string) (*core.Assistant, bool, error) {
	ctx := orm.NewContextWithParent(req.Context())
	return InternalGetAssistant(ctx, assistantID)
}

func InternalGetAssistant(ctx context.Context, assistantID string) (*core.Assistant, bool, error) {
	item := common.GeneralObjectCache.Get(core.AssistantCachePrimary, assistantID)
	var assistant *core.Assistant
	if item != nil && !item.Expired() {
		var ok bool
		if assistant, ok = item.Value().(*core.Assistant); ok {
			return assistant, true, nil
		}
	}
	assistant = &core.Assistant{}
	assistant.ID = assistantID

	ctx1 := orm.NewContextWithParent(ctx)
	ctx1.DirectAccess()
	exists, err := orm.GetV2(ctx1, assistant)
	if err != nil {
		return nil, exists, err
	}

	//expand datasource is the datasource is `*`
	if util.ContainsAnyInArray("*", assistant.Datasource.IDs) {
		ids, err := common.GetAllEnabledDatasourceIDs()
		if err != nil {
			panic(err)
		}
		assistant.Datasource.SetIDs(ids)
	}

	if util.ContainsAnyInArray("*", assistant.MCPConfig.IDs) {
		ids, err := common.GetAllEnabledMCPServerIDs()
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
		assistant.AnsweringModel.PromptConfig = &core.PromptConfig{PromptTemplate: common.GenerateAnswerPromptTemplate}
	} else if assistant.AnsweringModel.PromptConfig.PromptTemplate == "" {
		assistant.AnsweringModel.PromptConfig.PromptTemplate = common.GenerateAnswerPromptTemplate
	}

	switch assistant.Type {
	case core.AssistantTypeDeepThink:
		cfg := core.DeepThinkConfig{}
		buf := util.MustToJSONBytes(assistant.Config)
		util.MustFromJSONBytes(buf, &cfg)

		if cfg.IntentAnalysisModel.PromptConfig == nil {
			cfg.IntentAnalysisModel.PromptConfig = &core.PromptConfig{PromptTemplate: common.QueryIntentPromptTemplate}
		} else if cfg.IntentAnalysisModel.PromptConfig.PromptTemplate == "" {
			cfg.IntentAnalysisModel.PromptConfig.PromptTemplate = common.QueryIntentPromptTemplate
		}

		//assistant.Config = deepThinkCfg
		assistant.DeepThinkConfig = &cfg
	case core.AssistantTypeDeepResearch:
		// Deserialize the config
		userCfg := core.DeepResearchConfig{}
		buf := util.MustToJSONBytes(assistant.Config)
		util.MustFromJSONBytes(buf, &userCfg)
		// Validate the user config and merge it with default values
		if err = userCfg.Validate(); err != nil {
			return nil, exists, err
		}
		cfg := core.MergeDeepResearchConfig(&userCfg, core.DefaultDeepResearchConfig())

		assistant.DeepResearchConfig = cfg
	}

	if assistant.RolePrompt == "" {
		assistant.RolePrompt = "You are a personal AI assistant designed by Coco AI(https://coco.rs), the backend team is behind INFINI Labs(https://infinilabs.com)."
	}

	// Cache the assistant object
	common.GeneralObjectCache.Set(core.AssistantCachePrimary, assistantID, assistant, time.Duration(30)*time.Minute)
	return assistant, true, nil
}

var TotalAssistantsCacheKey = "total_assistants"

func ClearAssistantsCache() {
	common.GeneralObjectCache.Delete(core.AssistantCachePrimary, TotalAssistantsCacheKey)
}

func CountAssistants() (int64, error) {
	item := common.GeneralObjectCache.Get(core.AssistantCachePrimary, TotalAssistantsCacheKey)
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
		common.GeneralObjectCache.Set(core.AssistantCachePrimary, TotalAssistantsCacheKey, count, time.Duration(30)*time.Minute)
	}

	return count, err
}
