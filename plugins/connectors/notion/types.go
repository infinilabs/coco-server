/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package notion

// RichTextItem is a component of many block types.
type RichTextItem struct {
	PlainText string `json:"plain_text"`
}

// Block represents a generic content block in Notion.
type Block map[string]interface{}

// BlockChildrenResponse represents the response from the GET /v1/blocks/{block_id}/children endpoint
type BlockChildrenResponse struct {
	Object     string  `json:"object"`
	Results    []Block `json:"results"`
	NextCursor string  `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

// Type returns the value of the "type" key.
func (b Block) Type() string {
	if v, ok := b["type"]; ok {
		if result, ok := v.(string); ok {
			return result
		}
	}
	return ""
}

// GetRichTextSliceBy extracts a slice of RichTextItem from a given key.
func (b Block) GetRichTextSliceBy(key string) []RichTextItem {
	var result []RichTextItem

	// Safely get the value for the given key and assert it's a map.
	richTextMap, ok := b[key].(map[string]interface{})
	if !ok {
		return result
	}

	// Safely get the "rich_text" key and assert it's a slice.
	richTextSlice, ok := richTextMap["rich_text"].([]interface{})
	if !ok {
		return result
	}

	// Iterate through the slice, safely getting the plain text.
	for _, richTextItem := range richTextSlice {
		// Safely assert the item is a map and get "plain_text".
		itemMap, ok := richTextItem.(map[string]interface{})
		if !ok {
			continue
		}

		plainText, ok := itemMap["plain_text"].(string)
		if ok {
			result = append(result, RichTextItem{PlainText: plainText})
		}
	}

	return result
}
