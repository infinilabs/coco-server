package neo4j

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

func recordToMap(record *neo4j.Record) map[string]interface{} {
	payload := make(map[string]interface{})
	for i, key := range record.Keys {
		flattenValue(payload, key, record.Values[i])
	}
	return payload
}

func flattenValue(target map[string]interface{}, prefix string, value interface{}) {
	switch v := value.(type) {
	case dbtype.Node:
		target[prefix] = v.Props
		for k, val := range v.Props {
			flattenValue(target, fmt.Sprintf("%s.%s", prefix, k), val)
		}
		target[fmt.Sprintf("%s.element_id", prefix)] = v.ElementId
		if len(v.Labels) > 0 {
			target[fmt.Sprintf("%s.labels", prefix)] = v.Labels
		}
	case *dbtype.Node:
		target[prefix] = v.Props
		for k, val := range v.Props {
			flattenValue(target, fmt.Sprintf("%s.%s", prefix, k), val)
		}
		target[fmt.Sprintf("%s.element_id", prefix)] = v.ElementId
		if len(v.Labels) > 0 {
			target[fmt.Sprintf("%s.labels", prefix)] = v.Labels
		}
	case dbtype.Relationship:
		target[prefix] = v.Props
		for k, val := range v.Props {
			flattenValue(target, fmt.Sprintf("%s.%s", prefix, k), val)
		}
		target[fmt.Sprintf("%s.element_id", prefix)] = v.ElementId
		target[fmt.Sprintf("%s.start_element_id", prefix)] = v.StartElementId
		target[fmt.Sprintf("%s.end_element_id", prefix)] = v.EndElementId
		target[fmt.Sprintf("%s.type", prefix)] = v.Type
	case *dbtype.Relationship:
		target[prefix] = v.Props
		for k, val := range v.Props {
			flattenValue(target, fmt.Sprintf("%s.%s", prefix, k), val)
		}
		target[fmt.Sprintf("%s.element_id", prefix)] = v.ElementId
		target[fmt.Sprintf("%s.start_element_id", prefix)] = v.StartElementId
		target[fmt.Sprintf("%s.end_element_id", prefix)] = v.EndElementId
		target[fmt.Sprintf("%s.type", prefix)] = v.Type
	case dbtype.Path:
		target[prefix] = v
		for idx, node := range v.Nodes {
			flattenValue(target, fmt.Sprintf("%s.nodes[%d]", prefix, idx), node)
		}
		for idx, rel := range v.Relationships {
			flattenValue(target, fmt.Sprintf("%s.relationships[%d]", prefix, idx), rel)
		}
	case *dbtype.Path:
		target[prefix] = v
		for idx, node := range v.Nodes {
			flattenValue(target, fmt.Sprintf("%s.nodes[%d]", prefix, idx), node)
		}
		for idx, rel := range v.Relationships {
			flattenValue(target, fmt.Sprintf("%s.relationships[%d]", prefix, idx), rel)
		}
	case map[string]interface{}:
		target[prefix] = v
		for k, val := range v {
			flattenValue(target, fmt.Sprintf("%s.%s", prefix, k), val)
		}
	case []interface{}:
		target[prefix] = v
		for idx, item := range v {
			flattenValue(target, fmt.Sprintf("%s[%d]", prefix, idx), item)
		}
	default:
		target[prefix] = v
	}
}

func paramsForLogging(params map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{}, len(params))
	for k, v := range params {
		lowered := strings.ToLower(k)
		if strings.Contains(lowered, "password") || strings.Contains(lowered, "token") || strings.Contains(lowered, "secret") {
			sanitized[k] = "***"
			continue
		}
		sanitized[k] = v
	}
	return sanitized
}

func cloneParameters(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return map[string]interface{}{}
	}
	cloned := make(map[string]interface{}, len(src))
	for k, v := range src {
		cloned[k] = v
	}
	return cloned
}

func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		res, _ := strconv.ParseInt(val, 10, 64)
		return res
	default:
		return 0
	}
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case float32:
		return float64(val)
	case float64:
		return val
	case string:
		res, _ := strconv.ParseFloat(val, 64)
		return res
	default:
		return 0
	}
}

func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		res, err := strconv.ParseBool(val)
		if err == nil {
			return res
		}
		return false
	default:
		return false
	}
}
