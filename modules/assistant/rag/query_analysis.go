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
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/util"
	"regexp"
	"strings"
)

type QueryIntent struct {
	Category   string   `json:"category"`
	Intent     string   `json:"intent"`
	Query      []string `json:"query"`
	Keyword    []string `json:"keyword"`
	Suggestion []string `json:"suggestion"`
}

func QueryAnalysisFromString(str string) (*QueryIntent, error) {
	log.Trace("input:", str)
	jsonContent := extractJSON(str)
	obj := QueryIntent{}
	err := util.FromJSONBytes([]byte(jsonContent), &obj)
	if err != nil {
		return nil, err
	}
	return &obj, nil
}

var jsonBlockTag = regexp.MustCompile(`(?m)(.*?)\<JSON\>([\w\W]+)\<\/JSON\>(.*?)`)
var jsonMarkdownTag = regexp.MustCompile(`(?m)(.*?[\x60]{3,})json([\w\W]+)([\x60]{3,})(.*?)`)

func extractJSON(input string) string {
	matches := jsonMarkdownTag.FindAllStringSubmatch(input, -1)
	if len(matches) > 0 {
		if len(matches[0]) > 2 {
			return strings.TrimSpace(matches[0][2])
		}
	}

	matches = jsonBlockTag.FindAllStringSubmatch(input, -1)
	if len(matches) > 0 {
		if len(matches[0]) > 2 {
			return strings.TrimSpace(matches[0][2])
		}
	}

	return ""
}
