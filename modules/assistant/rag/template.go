// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package rag

import (
	"github.com/tmc/langchaingo/prompts"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/errors"
	"regexp"
)

// extractVariables parses a Go template string and returns a slice of unique variable names
// used in the {{.variable}} syntax.
func extractVariables(template string) []string {
	// Regular expression to match {{ .variable }} patterns
	re := regexp.MustCompile(`{{\s*\.\s*([a-zA-Z0-9_]+)\s*}}`)

	// Find all matches
	matches := re.FindAllStringSubmatch(template, -1)

	// Use a map to store unique variable names
	varsMap := make(map[string]struct{})
	for _, match := range matches {
		if len(match) > 1 {
			varsMap[match[1]] = struct{}{}
		}
	}

	// Convert map keys to a slice
	vars := make([]string, 0, len(varsMap))
	for v := range varsMap {
		vars = append(vars, v)
	}

	return vars
}

func GetPromptStringByTemplateArgs(cfg *common.ModelConfig, defaultTemplate string, requiredVars []string, inputValues map[string]any)(string,error)  {
	promptTemplate,err:= GetPromptTemplate(cfg,defaultTemplate,requiredVars,inputValues)
	if err!=nil{
		return "",err
	}
	promptValues, err := promptTemplate.FormatPrompt(inputValues)
	if err != nil {
		return "", err
	}

	return promptValues.String(),nil
}

func GetPromptTemplate(cfg *common.ModelConfig, defaultTemplate string, requiredVars []string, inputValues map[string]any) (*prompts.PromptTemplate, error) {
	template := defaultTemplate
	inputVars := requiredVars

	if cfg.PromptConfig != nil {
		if cfg.PromptConfig.PromptTemplate != "" {
			template = cfg.PromptConfig.PromptTemplate
		}

		if len(cfg.PromptConfig.InputVars) > 0 {
			inputVars = cfg.PromptConfig.InputVars
		}
	}

	variables := extractVariables(template)
	missingVars := map[string]interface{}{}
	for _, v := range variables {
		if _, exists := inputValues[v]; !exists {
			missingVars[v] = ""
		}
	}

	if len(missingVars) > 0 && len(requiredVars) > 0 {
		for _, v := range requiredVars {
			_, ok := missingVars[v]
			if ok {
				return nil, errors.Errorf("var [%v] required, but was not found", v)
			}
		}
	}

	prompt := prompts.NewPromptTemplate(template, inputVars)
	prompt.PartialVariables = missingVars //default value for missing variable

	return &prompt, nil
}
