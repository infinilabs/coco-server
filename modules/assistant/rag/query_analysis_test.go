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
	"infini.sh/framework/core/util"
	"testing"
)

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "valid json block",
			input: `Some text
			<JSON>
			{
				"category": "Test"
			}
			</JSON>
			More text`,
			expected: `{
				"category": "Test"
			}`,
		},
		{
			name:  "valid json markdown block",
			input: "Some text ```json\n\t\t\t{\n\t\t\t\t\"category\": \"Test\"\n\t\t\t}\n\t\t\t``` More text",
			expected: `{
				"category": "Test"
			}`,
		},
		{
			name:     "multiple json blocks",
			input:    `<JSON>{"a": 1}</JSON>`,
			expected: `{"a": 1}`,
		},
		{
			name:     "no json block",
			input:    "Just some regular text",
			expected: "",
		},
		{
			name: "malformed tags",
			input: `<JSON
			{"invalid": "tags"}
			</JSON>`,
			expected: "",
		},
		{
			name: "empty json content",
			input: `<JSON>
			</JSON>`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("Expected:\n%v\n\nGot:\n%v", tt.expected, result)
			}
		})
	}
}

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    QueryIntent
		expectError bool
	}{
		{
			name: "valid json",
			input: `{
				"category": "Greeting",
				"intent": "Test",
				"query": ["test"],
				"keyword": ["test"],
				"suggestion": ["test"]
			}`,
			expected: QueryIntent{
				Category:   "Greeting",
				Intent:     "Test",
				Query:      []string{"test"},
				Keyword:    []string{"test"},
				Suggestion: []string{"test"},
			},
			expectError: false,
		},
		{
			name: "invalid json structure",
			input: `{
				"category": "Greeting",
				"intent": 123,
				"query": "not an array"
			}`,
			expectError: true,
		},
		{
			name: "unicode characters",
			input: `{
				"category": "问候",
				"intent": "测试",
				"query": ["你好"],
				"keyword": ["中文"],
				"suggestion": ["欢迎"]
			}`,
			expected: QueryIntent{
				Category:   "问候",
				Intent:     "测试",
				Query:      []string{"你好"},
				Keyword:    []string{"中文"},
				Suggestion: []string{"欢迎"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result = QueryIntent{}
			err := util.FromJson(tt.input, &result)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Intent != tt.expected.Intent {
				t.Errorf("Expected:\n%+v\n\nGot:\n%+v", tt.expected, result)
			}
		})
	}
}
