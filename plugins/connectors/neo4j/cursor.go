package neo4j

import (
	"cmp"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"infini.sh/coco/plugins/connectors"
)

type cursorSnapshot struct {
	stored   *connectors.StoredCursor
	property interface{}
	tie      interface{}
}

type cursorFactory struct {
	propertyType string
}

func (c *cursorFactory) fromValue(property interface{}, tie interface{}) (*cursorSnapshot, error) {
	storedProperty, normalizedProperty, err := normalizeCursorValue(property, c.propertyType)
	if err != nil {
		return nil, err
	}

	var storedTie *connectors.StoredCursorValue
	var normalizedTie interface{}
	if tie != nil {
		storedTie, normalizedTie, err = normalizeCursorValue(tie, "")
		if err != nil {
			return nil, err
		}
	}

	return &cursorSnapshot{
		stored:   &connectors.StoredCursor{Property: *storedProperty, Tie: storedTie},
		property: normalizedProperty,
		tie:      normalizedTie,
	}, nil
}

func (c *cursorFactory) fromStored(stored *connectors.StoredCursor) (*cursorSnapshot, error) {
	if stored == nil {
		return nil, nil
	}

	property, err := decodeCursorValue(&stored.Property, c.propertyType)
	if err != nil {
		return nil, err
	}

	var tie interface{}
	if stored.Tie != nil {
		tie, err = decodeCursorValue(stored.Tie, "")
		if err != nil {
			return nil, err
		}
	}

	return &cursorSnapshot{
		stored:   stored,
		property: property,
		tie:      tie,
	}, nil
}

func (c *cursorFactory) fromResume(raw string) (*cursorSnapshot, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	valueStr := strings.TrimSpace(raw)
	stored := &connectors.StoredCursorValue{Value: valueStr}
	if c.propertyType != "" {
		stored.Type = c.propertyType
	} else {
		stored.Type = "string"
	}
	decoded, err := decodeCursorValue(stored, c.propertyType)
	if err != nil {
		return nil, err
	}
	normalizedStored, normalizedValue, err := normalizeCursorValue(decoded, c.propertyType)
	if err != nil {
		return nil, err
	}
	return &cursorSnapshot{
		stored:   &connectors.StoredCursor{Property: *normalizedStored},
		property: normalizedValue,
	}, nil
}

func normalizeCursorValue(value interface{}, propertyType string) (*connectors.StoredCursorValue, interface{}, error) {
	if value == nil {
		return nil, nil, errors.New("cursor value is nil")
	}

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

	if iv, ok := tryInt64(value); ok {
		stored, normalized := storeInt64(iv)
		return stored, normalized, nil
	}

	if fv, ok := tryFloat64(value); ok {
		stored, normalized := storeFloat64(fv)
		return stored, normalized, nil
	}

	if bv, ok := value.(bool); ok {
		stored, normalized := storeBool(bv)
		return stored, normalized, nil
	}

	strVal := stringFromValue(value)
	trimmed := strings.TrimSpace(strVal)
	hint := normalizePropertyType(propertyType)

	switch hint {
	case "int":
		stored, normalized := storeInt64(toInt64(trimmed))
		return stored, normalized, nil
	case "float":
		stored, normalized := storeFloat64(toFloat64(trimmed))
		return stored, normalized, nil
	case "datetime":
		t, err := parseTimeString(trimmed)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to convert value %v to datetime", value)
		}
		stored, normalized := storeTime(t)
		return stored, normalized, nil
	default:
		stored, normalized := storeString(strVal)
		return stored, normalized, nil
	}
}

func decodeCursorValue(value *connectors.StoredCursorValue, overrideType string) (interface{}, error) {
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

var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999 -0700 MST",
	"2006-01-02 15:04:05 -0700 MST",
	time.RFC1123Z,
	time.RFC1123,
	time.DateTime,
}

func parseTimeString(input string) (time.Time, error) {
	s := strings.TrimSpace(input)
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time value %q", input)
}

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

func compareCursor(a, b *cursorSnapshot, propertyType string) int {
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
	if typ == "" && a.stored != nil {
		typ = a.stored.Property.Type
	}
	diff := compareValues(a.property, b.property, typ)
	if diff != 0 {
		return diff
	}

	switch at := a.tie.(type) {
	case string:
		bt, _ := b.tie.(string)
		return strings.Compare(at, bt)
	case int64:
		bt, _ := b.tie.(int64)
		return cmp.Compare(at, bt)
	case float64:
		bt, _ := b.tie.(float64)
		return cmp.Compare(at, bt)
	case time.Time:
		bt, _ := b.tie.(time.Time)
		if at.Equal(bt) {
			return 0
		}
		if at.Before(bt) {
			return -1
		}
		return 1
	default:
		if a.tie == nil && b.tie == nil {
			return 0
		}
		as := fmt.Sprintf("%v", a.tie)
		bs := fmt.Sprintf("%v", b.tie)
		return strings.Compare(as, bs)
	}
}

func compareValues(a, b interface{}, propertyType string) int {
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

func normalizePropertyType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "int", "integer":
		return "int"
	case "float", "double":
		return "float"
	case "datetime", "time", "timestamp":
		return "datetime"
	default:
		return "string"
	}
}
