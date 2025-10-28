package mongodb

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"infini.sh/coco/modules/common"
	cmn "infini.sh/coco/plugins/connectors/common"
)

// bsonToDocument converts a MongoDB BSON document to a common.Document using field mapping if enabled
func bsonToDocument(doc bson.M, cfg *Config, datasource *common.DataSource) (*common.Document, error) {
	// Normalize all BSON values to standard Go types
	payload := make(map[string]interface{})
	for key, value := range doc {
		payload[key] = normalizeBSONValue(value)
	}

	// Create the document with source information
	result := &common.Document{}
	result.Payload = payload

	// Set System and Source from datasource
	result.System = datasource.System
	result.Source = common.DataSourceReference{
		ID:   datasource.ID,
		Type: "connector",
		Name: datasource.Name,
	}

	// If field mapping is enabled, use the Transformer
	if cfg.FieldMapping.Enabled && cfg.FieldMapping.Mapping != nil {
		transformer := cmn.Transformer{
			Payload: payload,
			Visited: make(map[string]bool),
		}
		transformer.Transform(result, cfg.FieldMapping.Mapping)
	}

	return result, nil
}

// normalizeBSONValue converts MongoDB BSON types to standard Go types
func normalizeBSONValue(value interface{}) interface{} {
	switch v := value.(type) {
	case primitive.ObjectID:
		return v.Hex() // Convert ObjectID to hex string
	case primitive.DateTime:
		return v.Time() // Convert to time.Time
	case primitive.Binary:
		return v.Data // Extract binary data
	case primitive.Decimal128:
		return v.String() // Convert Decimal128 to string
	case primitive.Timestamp:
		return time.Unix(int64(v.T), 0) // Convert timestamp to time.Time
	case primitive.Regex:
		return v.Pattern // Extract regex pattern
	case primitive.JavaScript:
		return string(v) // Convert JavaScript code to string
	case primitive.Symbol:
		return string(v) // Convert symbol to string
	case primitive.A: // BSON array
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = normalizeBSONValue(item)
		}
		return result
	case []interface{}: // Generic slice
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = normalizeBSONValue(item)
		}
		return result
	case []string: // String array - preserve as-is
		return v
	case bson.M: // BSON document (nested)
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = normalizeBSONValue(val)
		}
		return result
	case map[string]interface{}: // Generic map
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = normalizeBSONValue(val)
		}
		return result
	default:
		return v // Return as-is for standard types
	}
}
