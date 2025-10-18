package feishu

import (
	"strconv"
	"strings"
	"time"
)

// getString safely extracts a string value from a map
func getString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getBool safely extracts a boolean value from a map
func getBool(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key]; ok && v != nil {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// getTime parses various time formats and returns a time.Time
// Supports RFC3339, ISO formats, and Unix timestamps (seconds/milliseconds/microseconds/nanoseconds)
func getTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}

	// Try standard layouts first
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}

	// Try parsing as Unix timestamp (numeric string)
	if isNumeric(s) {
		if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
			return parseUnixTimestamp(ts, len(s))
		}
	}

	return time.Time{}
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}

// parseUnixTimestamp converts Unix timestamp to time.Time based on its length
func parseUnixTimestamp(ts int64, length int) time.Time {
	var sec, nsec int64
	switch {
	case length <= 10: // seconds
		sec = ts
		nsec = 0
	case length <= 13: // milliseconds
		sec = ts / 1_000
		nsec = (ts % 1_000) * int64(time.Millisecond)
	case length <= 16: // microseconds
		sec = ts / 1_000_000
		nsec = (ts % 1_000_000) * int64(time.Microsecond)
	default: // nanoseconds
		sec = ts / 1_000_000_000
		nsec = ts % 1_000_000_000
	}
	return time.Unix(sec, nsec)
}

// getIcon returns the appropriate icon for a document type
func getIcon(docType string) string {
	switch docType {
	case "doc", "sheet", "slides", "mindnote", "bitable", "file", "docx":
		return docType
	default:
		return "default"
	}
}
