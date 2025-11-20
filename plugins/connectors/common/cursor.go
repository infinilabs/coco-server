package common

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"infini.sh/coco/plugins/connectors"
)

// CursorWatermark represents a point-in-time cursor state with both stored and normalized values
type CursorWatermark struct {
	Stored   *connectors.StoredCursor
	Property interface{} // Normalized property value for comparison
	Tie      interface{} // Normalized tie-breaker value for comparison
}

// CursorSerializer creates and manages cursor snapshots with type-safe value handling
type CursorSerializer struct {
	PropertyType string // Expected type: "int", "float", "datetime", "string", "bool"
}

// NewCursorSerializer creates a new cursor factory with the specified property type
func NewCursorSerializer(propertyType string) *CursorSerializer {
	return &CursorSerializer{
		PropertyType: NormalizePropertyType(propertyType),
	}
}

// FromValue creates a cursor snapshot from database values
// property: primary sort value, tie: secondary sort value (tie-breaker)
func (c *CursorSerializer) FromValue(property interface{}, tie interface{}) (*CursorWatermark, error) {
	storedProperty, normalizedProperty, err := NormalizeCursorValue(property, c.PropertyType)
	if err != nil {
		return nil, err
	}

	var storedTie *connectors.StoredCursorValue
	var normalizedTie interface{}
	if tie != nil {
		storedTie, normalizedTie, err = NormalizeCursorValue(tie, "")
		if err != nil {
			return nil, err
		}
	}

	return &CursorWatermark{
		Stored:   &connectors.StoredCursor{Property: *storedProperty, Tie: storedTie},
		Property: normalizedProperty,
		Tie:      normalizedTie,
	}, nil
}

// FromStored creates a cursor snapshot from a persisted StoredCursor
func (c *CursorSerializer) FromStored(stored *connectors.StoredCursor) (*CursorWatermark, error) {
	if stored == nil {
		return nil, nil
	}

	property, err := DecodeCursorValue(&stored.Property, c.PropertyType)
	if err != nil {
		return nil, err
	}

	var tie interface{}
	if stored.Tie != nil {
		tie, err = DecodeCursorValue(stored.Tie, "")
		if err != nil {
			return nil, err
		}
	}

	return &CursorWatermark{
		Stored:   stored,
		Property: property,
		Tie:      tie,
	}, nil
}

// FromResume creates a cursor snapshot from a manual resume string
func (c *CursorSerializer) FromResume(raw string) (*CursorWatermark, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	valueStr := strings.TrimSpace(raw)
	stored := &connectors.StoredCursorValue{Value: valueStr}
	if c.PropertyType != "" {
		stored.Type = c.PropertyType
	} else {
		stored.Type = "string"
	}
	decoded, err := DecodeCursorValue(stored, c.PropertyType)
	if err != nil {
		return nil, err
	}
	normalizedStored, normalizedValue, err := NormalizeCursorValue(decoded, c.PropertyType)
	if err != nil {
		return nil, err
	}
	return &CursorWatermark{
		Stored:   &connectors.StoredCursor{Property: *normalizedStored},
		Property: normalizedValue,
	}, nil
}

// CompareCursors compares two cursor snapshots and returns:
// -1 if a < b, 0 if a == b, 1 if a > b
func CompareCursors(a, b *CursorWatermark, propertyType string) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	typ := propertyType
	if typ == "" && a.Stored != nil {
		typ = a.Stored.Property.Type
	}
	diff := CompareValues(a.Property, b.Property, typ)
	if diff != 0 {
		return diff
	}

	// If properties are equal, compare tie-breakers
	switch at := a.Tie.(type) {
	case string:
		bt, _ := b.Tie.(string)
		return strings.Compare(at, bt)
	case int64:
		bt, _ := b.Tie.(int64)
		return cmp.Compare(at, bt)
	case float64:
		bt, _ := b.Tie.(float64)
		return cmp.Compare(at, bt)
	case time.Time:
		bt, _ := b.Tie.(time.Time)
		if at.Equal(bt) {
			return 0
		}
		if at.Before(bt) {
			return -1
		}
		return 1
	default:
		if a.Tie == nil && b.Tie == nil {
			return 0
		}
		as := fmt.Sprintf("%v", a.Tie)
		bs := fmt.Sprintf("%v", b.Tie)
		return strings.Compare(as, bs)
	}
}

// NormalizeCursorValue converts a raw value to a typed StoredCursorValue and normalized Go value
func NormalizeCursorValue(value interface{}, propertyType string) (*connectors.StoredCursorValue, interface{}, error) {
	if value == nil {
		return nil, nil, errors.New("cursor value is nil")
	}

	// Handle time.Time directly
	switch v := value.(type) {
	case time.Time:
		stored, normalized := storeTime(v)
		return stored, normalized, nil
	case *time.Time:
		if v == nil {
			return nil, nil, errors.New("cursor value is nil")
		}
		stored, normalized := storeTime(*v)
		return stored, normalized, nil
	}

	// Try numeric types - but check property type hint first
	hint := NormalizePropertyType(propertyType)

	if iv, ok := tryInt64(value); ok {
		// If property type is datetime and value is int, treat as Unix timestamp (milliseconds)
		if hint == "datetime" {
			t := time.UnixMilli(iv)
			stored, normalized := storeTime(t)
			return stored, normalized, nil
		}
		stored, normalized := storeInt64(iv)
		return stored, normalized, nil
	}

	if fv, ok := tryFloat64(value); ok {
		// If property type is datetime and value is float, treat as Unix timestamp (seconds with fractional)
		if hint == "datetime" {
			sec := int64(fv)
			nsec := int64((fv - float64(sec)) * 1e9)
			t := time.Unix(sec, nsec)
			stored, normalized := storeTime(t)
			return stored, normalized, nil
		}
		stored, normalized := storeFloat64(fv)
		return stored, normalized, nil
	}

	if bv, ok := value.(bool); ok {
		stored, normalized := storeBool(bv)
		return stored, normalized, nil
	}

	// Fall back to string conversion with type hints
	strVal := stringFromValue(value)
	trimmed := strings.TrimSpace(strVal)

	switch hint {
	case "int":
		stored, normalized := storeInt64(toInt64(trimmed))
		return stored, normalized, nil
	case "float":
		stored, normalized := storeFloat64(toFloat64(trimmed))
		return stored, normalized, nil
	case "datetime":
		// Try to parse as Unix timestamp first (string representation of milliseconds)
		if ts, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			t := time.UnixMilli(ts)
			stored, normalized := storeTime(t)
			return stored, normalized, nil
		}
		// Try to parse as datetime string
		t, err := ParseTimeString(trimmed)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to convert value %v to datetime: %w", value, err)
		}
		stored, normalized := storeTime(t)
		return stored, normalized, nil
	default:
		stored, normalized := storeString(strVal)
		return stored, normalized, nil
	}
}

// DecodeCursorValue decodes a StoredCursorValue back to a Go value
func DecodeCursorValue(value *connectors.StoredCursorValue, overrideType string) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	valueType := value.Type
	if overrideType != "" {
		valueType = overrideType
	}

	switch valueType {
	case "int":
		iv, err := strconv.ParseInt(value.Value, 10, 64)
		if err != nil {
			return nil, err
		}
		return iv, nil
	case "float":
		fv, err := strconv.ParseFloat(value.Value, 64)
		if err != nil {
			return nil, err
		}
		return fv, nil
	case "datetime":
		t, err := time.Parse(time.RFC3339Nano, value.Value)
		if err != nil {
			return nil, err
		}
		return t, nil
	case "bool":
		bv, err := strconv.ParseBool(value.Value)
		if err != nil {
			return nil, err
		}
		return bv, nil
	default:
		return value.Value, nil
	}
}

// CompareValues compares two values based on their type
func CompareValues(a, b interface{}, propertyType string) int {
	switch propertyType {
	case "int":
		return cmp.Compare(toInt64(a), toInt64(b))
	case "float":
		return cmp.Compare(toFloat64(a), toFloat64(b))
	case "datetime":
		at, _ := a.(time.Time)
		bt, _ := b.(time.Time)
		if at.Equal(bt) {
			return 0
		}
		if at.Before(bt) {
			return -1
		}
		return 1
	case "bool":
		ab := toBool(a)
		bb := toBool(b)
		if ab == bb {
			return 0
		}
		if !ab && bb {
			return -1
		}
		return 1
	default:
		as := fmt.Sprintf("%v", a)
		bs := fmt.Sprintf("%v", b)
		return strings.Compare(as, bs)
	}
}

// NormalizePropertyType normalizes type strings to canonical forms
func NormalizePropertyType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "int", "integer", "long":
		return "int"
	case "float", "double", "decimal":
		return "float"
	case "datetime", "time", "timestamp", "date":
		return "datetime"
	case "bool", "boolean":
		return "bool"
	default:
		return "string"
	}
}

// ParseTimeString attempts to parse a time string using multiple common formats
var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999 -0700 MST",
	"2006-01-02 15:04:05 -0700 MST",
	time.RFC1123Z,
	time.RFC1123,
	time.DateTime,
	"2006-01-02",
}

func ParseTimeString(input string) (time.Time, error) {
	s := strings.TrimSpace(input)
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time value %q", input)
}

// Internal helper functions for value storage and conversion

func storeString(s string) (*connectors.StoredCursorValue, string) {
	return &connectors.StoredCursorValue{Type: "string", Value: s}, s
}

func storeBool(b bool) (*connectors.StoredCursorValue, bool) {
	return &connectors.StoredCursorValue{Type: "bool", Value: strconv.FormatBool(b)}, b
}

func storeInt64(iv int64) (*connectors.StoredCursorValue, int64) {
	return &connectors.StoredCursorValue{Type: "int", Value: strconv.FormatInt(iv, 10)}, iv
}

func storeFloat64(fv float64) (*connectors.StoredCursorValue, float64) {
	return &connectors.StoredCursorValue{Type: "float", Value: strconv.FormatFloat(fv, 'f', -1, 64)}, fv
}

func storeTime(t time.Time) (*connectors.StoredCursorValue, time.Time) {
	utc := t.UTC()
	return &connectors.StoredCursorValue{Type: "datetime", Value: utc.Format(time.RFC3339Nano)}, utc
}

func tryInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	default:
		return 0, false
	}
}

func tryFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func stringFromValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int8:
		return int64(val)
	case int16:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	case uint:
		return int64(val)
	case uint8:
		return int64(val)
	case uint16:
		return int64(val)
	case uint32:
		return int64(val)
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
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return toInt64(val) != 0
	default:
		return false
	}
}

// CursorStateManager provides helper methods for loading and saving cursor state
type CursorStateManager struct {
	ConnectorID  string
	DatasourceID string
	Serializer   *CursorSerializer
	StateStore   *connectors.SyncStateStore
}

// Load loads cursor from state store with property validation
// Returns nil cursor if property changed (automatic reset)
func (p *CursorStateManager) Load(ctx context.Context, currentProperty string) (*CursorWatermark, error) {
	state, err := p.StateStore.Load(ctx, p.ConnectorID, p.DatasourceID)
	if err != nil {
		if err.Error() == "record not found" {
			// Not an error - just no cursor saved yet
			return nil, nil
		}
		return nil, err
	}
	if state == nil || state.Cursor == nil {
		return nil, nil
	}

	// Validate property hasn't changed
	savedProperty := strings.TrimSpace(state.Property)
	currentProp := strings.TrimSpace(currentProperty)
	if currentProp == "" || savedProperty == "" {
		return nil, nil
	}
	if savedProperty != currentProp {
		// Property changed - reset cursor
		return nil, nil
	}

	return p.Serializer.FromStored(state.Cursor)
}

// Save persists cursor state to the state store
func (p *CursorStateManager) Save(ctx context.Context, property string, snapshot *CursorWatermark) error {
	if snapshot == nil || snapshot.Stored == nil {
		return nil
	}

	state := &connectors.SyncState{
		ConnectorID:  p.ConnectorID,
		DatasourceID: p.DatasourceID,
		Mode:         ModePropertyWatermark,
		Property:     property,
		Cursor:       snapshot.Stored,
	}

	return p.StateStore.Save(ctx, state)
}

func (p *CursorStateManager) LoadWithFallback(ctx context.Context, syncCfg IncrementalConfig) (*CursorWatermark, error) {
	storedCursor, err := p.Load(ctx, syncCfg.Property)
	if err != nil {
		return nil, err
	}

	// If we have a stored cursor, use it
	if storedCursor != nil {
		return storedCursor, nil
	}

	// Fall back to manual resume_from if no stored cursor
	if syncCfg.ResumeFrom != "" {
		snapshot, e := p.Serializer.FromResume(syncCfg.ResumeFrom)
		if e != nil {
			return nil, e
		}
		return snapshot, nil
	}

	// No cursor available
	return nil, nil
}
