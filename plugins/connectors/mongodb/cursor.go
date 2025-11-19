package mongodb

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"infini.sh/coco/plugins/connectors"
	cmn "infini.sh/coco/plugins/connectors/common"
)

// extractCursor extracts cursor values from a MongoDB document
func (s *scanner) extractCursor(doc bson.M) (*cmn.CursorWatermark, error) {
	// Extract property value
	propertyValue, ok := doc[s.config.Incremental.Property]
	if !ok {
		return nil, fmt.Errorf("field %s not found in document", s.config.Incremental.Property)
	}

	// Extract tie-breaker value if configured
	var tieValue interface{}
	if s.config.Incremental.TieBreaker != "" {
		tieValue, ok = doc[s.config.Incremental.TieBreaker]
		if !ok {
			return nil, fmt.Errorf("tie-breaker field %s not found in document", s.config.Incremental.TieBreaker)
		}
	}

	// Create cursor snapshot with BSON type preservation
	return s.createCursorSnapshot(propertyValue, tieValue)
}

// createCursorSnapshot creates a cursor snapshot preserving MongoDB BSON types
func (s *scanner) createCursorSnapshot(propertyValue, tieValue interface{}) (*cmn.CursorWatermark, error) {
	// Normalize property value and preserve BSON type
	storedProperty, normalizedProperty, err := s.normalizeBSONForCursor(propertyValue, s.config.GetPropertyType())
	if err != nil {
		return nil, err
	}

	// Normalize tie-breaker value if present
	var storedTie *connectors.StoredCursorValue
	var normalizedTie interface{}
	if tieValue != nil {
		storedTie, normalizedTie, err = s.normalizeBSONForCursor(tieValue, "")
		if err != nil {
			return nil, err
		}
	}

	return &cmn.CursorWatermark{
		Stored:   &connectors.StoredCursor{Property: *storedProperty, Tie: storedTie},
		Property: normalizedProperty,
		Tie:      normalizedTie,
	}, nil
}

// normalizeBSONForCursor converts BSON value to StoredCursorValue preserving raw type
func (s *scanner) normalizeBSONForCursor(value interface{}, propertyType string) (*connectors.StoredCursorValue, interface{}, error) {
	if value == nil {
		return nil, nil, fmt.Errorf("cursor value is nil")
	}

	var rawType string
	var normalizedValue interface{}
	var storedValue connectors.StoredCursorValue

	// Handle MongoDB-specific BSON types
	switch v := value.(type) {
	case primitive.ObjectID:
		rawType = "ObjectId" // MongoDB uses ObjectId() not ObjectID()
		normalizedValue = v.Hex()
		storedValue = connectors.StoredCursorValue{
			Type:    "string",
			Value:   v.Hex(),
			RawType: rawType,
		}
		return &storedValue, normalizedValue, nil

	case primitive.DateTime:
		rawType = "DateTime"
		t := v.Time()
		normalizedValue = t
		storedValue = connectors.StoredCursorValue{
			Type:    "datetime",
			Value:   t.UTC().Format(time.RFC3339Nano),
			RawType: rawType,
		}
		return &storedValue, normalizedValue, nil
	}

	// For other types (including time.Time), use the common normalization
	return cmn.NormalizeCursorValue(value, propertyType)
}
