/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package langchain

import (
	"net/http"
	"net/http/httputil"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/global"
)

type LoggingRoundTripper struct {
	original http.RoundTripper
}

func (lrt *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Dump and log request
	dump, err := httputil.DumpRequestOut(req, true)
	if err == nil {
		log.Info("=== API Request ===")
		log.Info(string(dump))
	}
	return lrt.original.RoundTrip(req)
}

func SimplyGetLLM(providerID, modelName string, keepalive string) (llms.Model, error) {

	modelProvider, err := common.GetModelProvider(providerID)
	if err != nil {
		return nil, err
	}

	llm := GetLLM(modelProvider.BaseURL, modelProvider.APIType, modelName, modelProvider.APIKey, keepalive)

	return llm, nil
}
func GetLLMByConfig(model core.ModelConfig) (llms.Model, error) {

	modelProvider, err := common.GetModelProvider(model.ProviderID)
	if err != nil {
		return nil, err
	}

	llm := GetLLM(modelProvider.BaseURL, modelProvider.APIType, model.Name, modelProvider.APIKey, model.Keepalive)

	return llm, nil
}

func GetLLM(endpoint, apiType, model, token string, keepalive string) llms.Model {
	if model == "" {
		panic("model is empty")
	}

	log.Debug("use model:", model, ",type:", apiType)

	if apiType == common.OLLAMA {
		llm, err := ollama.New(
			ollama.WithServerURL(endpoint),
			ollama.WithModel(model),
			ollama.WithKeepAlive(keepalive))
		if err != nil {
			panic(err)
		}
		return llm

	} else {

		var llm llms.Model
		var err error

		if global.Env().IsDebug {
			customClient := &http.Client{
				Transport: &LoggingRoundTripper{original: http.DefaultTransport},
			}
			llm, err = openai.New(
				openai.WithHTTPClient(customClient),
				openai.WithToken(token),
				openai.WithBaseURL(endpoint),
				openai.WithModel(model),
				openai.WithEmbeddingModel(model),
			)
		} else {
			llm, err = openai.New(
				openai.WithToken(token),
				openai.WithBaseURL(endpoint),
				openai.WithModel(model),
				openai.WithEmbeddingModel(model),
			)
		}

		if err != nil {
			panic(err)
		}
		return llm
	}
}

func GetTemperature(model *core.ModelConfig, defaultValue float64) float64 {
	if model.Settings.Temperature > 0 {
		return model.Settings.Temperature
	}
	return defaultValue
}

func GetMaxLength(model *core.ModelConfig, defaultValue int) int {
	if model.Settings.MaxLength > 0 {
		return model.Settings.MaxLength
	}
	return defaultValue
}

func GetMaxTokens(model *core.ModelConfig, defaultValue int) int {
	if model.Settings.MaxTokens > 0 {
		return model.Settings.MaxTokens
	}
	return defaultValue
}

func GetLLOptions(model *core.ModelConfig) []llms.CallOption {
	options := []llms.CallOption{}
	maxTokens := GetMaxTokens(model, 8192)
	temperature := GetTemperature(model, 0.9)
	//maxLength := GetMaxLength(model, 0)
	options = append(options, llms.WithMaxTokens(maxTokens))
	//options = append(options, llms.WithMaxLength(maxLength))
	options = append(options, llms.WithTemperature(temperature))
	// Check if the model supports reasoning and reasoning is enabled in settings
	if common.ModelSupportsReasoning(model.ProviderID, model.Name) && model.Settings.Reasoning {
		options = append(options, llms.WithThinking(&llms.ThinkingConfig{
			Mode:           llms.ThinkingModeAuto,
			StreamThinking: true,
		}))
	}
	return options
}
