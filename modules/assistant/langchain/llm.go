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

func GetTemperature(model *core.ModelConfig, modelProvider *core.ModelProvider, defaultValue float64) float64 {
	temperature := 0.0
	if model.Settings.Temperature > 0 {
		temperature = model.Settings.Temperature
	}
	if temperature == 0 {
		for _, m := range modelProvider.Models {
			if m.Name == model.Name {
				if m.Settings.Temperature > 0 {
					temperature = m.Settings.Temperature
				}
				break
			}
		}
	}
	if temperature == 0 {
		temperature = defaultValue
	}
	return temperature
}

func GetMaxLength(model *core.ModelConfig, modelProvider *core.ModelProvider, defaultValue int) int {
	maxLength := 0
	if model.Settings.MaxLength > 0 {
		maxLength = model.Settings.MaxLength
	}
	if maxLength == 0 {
		for _, m := range modelProvider.Models {
			if m.Name == model.Name {
				if m.Settings.MaxLength > 0 {
					maxLength = m.Settings.MaxLength
				}
				break
			}
		}
	}
	if maxLength == 0 {
		maxLength = defaultValue
	}
	return maxLength
}

func GetMaxTokens(model *core.ModelConfig, modelProvider *core.ModelProvider, defaultValue int) int {
	var maxTokens int = 0
	if model.Settings.MaxTokens > 0 {
		maxTokens = model.Settings.MaxTokens
	}
	if maxTokens == 0 {
		for _, m := range modelProvider.Models {
			if m.Name == model.Name {
				if m.Settings.MaxTokens > 0 {
					maxTokens = m.Settings.MaxTokens
				}
				break
			}
		}
	}
	if maxTokens == 0 {
		maxTokens = defaultValue
	}
	return maxTokens
}
