/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"infini.sh/framework/core/global"
)

func (p *Plugin) safeConvertToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case primitive.ObjectID:
		return v.Hex()
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case time.Time:
		return v.Format(time.RFC3339)
	case primitive.DateTime:
		return v.Time().Format(time.RFC3339)
	case primitive.Timestamp:
		return time.Unix(int64(v.T), 0).Format(time.RFC3339)
	case []interface{}:
		// Convert array to JSON string
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", v)
	case map[string]interface{}:
		// Convert object to JSON string
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", v)
	default:
		// Try JSON serialization as fallback
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", v)
	}
}

func (p *Plugin) convertToStringSlice(value interface{}) []string {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		var result []string
		for _, item := range v {
			result = append(result, p.safeConvertToString(item))
		}
		return result
	case string:
		// If it's a single string, treat as one tag
		return []string{v}
	default:
		// Convert to string and treat as single tag
		return []string{p.safeConvertToString(v)}
	}
}

func (p *Plugin) convertToTime(value interface{}) *time.Time {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		return &v
	case primitive.DateTime:
		t := v.Time()
		return &t
	case primitive.Timestamp:
		t := time.Unix(int64(v.T), 0)
		return &t
	case int64:
		// Unix timestamp
		t := time.Unix(v, 0)
		return &t
	case string:
		// Try to parse various time formats
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return &t
			}
		}
	}

	return nil
}

func (p *Plugin) shouldStop() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.ctx == nil {
		return true
	}

	select {
	case <-p.ctx.Done():
		return true
	default:
		return global.ShuttingDown()
	}
}
