/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"fmt"
	"infini.sh/coco/core"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
)

type Transformer struct {
	Payload map[string]interface{}
	Visited map[string]bool
}

func (t *Transformer) Transform(doc *core.Document, m *Mapping) {
	doc.ID = t.getString(m.ID)
	doc.Title = t.getString(m.Title)
	doc.URL = t.getString(m.URL)
	doc.Summary = t.getString(m.Summary)
	doc.Content = t.getString(m.Content)
	doc.Icon = t.getString(m.Icon)
	doc.Category = t.getString(m.Category)
	doc.Subcategory = t.getString(m.Subcategory)
	doc.Created = t.getTime(m.Created)
	doc.Updated = t.getTime(m.Updated)
	doc.Cover = t.getString(m.Cover)
	doc.Type = t.getString(m.Type)
	doc.Lang = t.getString(m.Lang)
	doc.Thumbnail = t.getString(m.Thumbnail)
	doc.Tags = t.getStringSlice(m.Tags)
	doc.Size = t.getInt(m.Size)

	owner := &core.UserInfo{
		UserAvatar: t.getString(m.Owner.Avatar),
		UserName:   t.getString(m.Owner.UserName),
		UserID:     t.getString(m.Owner.UserID),
	}

	if !isUserEmpty(owner) {
		doc.Owner = owner
	}

	doc.Metadata = make(map[string]interface{})

	for _, p := range m.Metadata {
		if p.Name != "" {
			doc.Metadata[p.Name] = t.getRaw(p.GetValue())
		}
	}
	doc.Payload = make(map[string]interface{})
	for _, p := range m.Payload {
		if p.Name != "" {
			doc.Payload[p.Name] = t.getRaw(p.GetValue())
		}
	}

	doc.LastUpdatedBy = &core.EditorInfo{
		UpdatedAt: t.getTime(m.LastUpdatedBy.Timestamp),
	}

	user := &core.UserInfo{
		UserAvatar: t.getString(m.LastUpdatedBy.UserInfo.Avatar),
		UserName:   t.getString(m.LastUpdatedBy.UserInfo.UserName),
		UserID:     t.getString(m.LastUpdatedBy.UserInfo.UserID),
	}
	if !isUserEmpty(user) {
		doc.LastUpdatedBy.UserInfo = user
	}

	// Append unvisited payload to doc's payload
	for k, v := range t.Payload {
		if !t.Visited[k] {
			doc.Payload[k] = v
		}
	}
}

func (t *Transformer) getString(column string) string {
	v, ok := t.Payload[column]
	if !ok || v == nil {
		return ""
	}
	t.Visited[column] = true
	if v, ok := v.([]uint8); ok {
		return string(v)
	}
	return fmt.Sprintf("%v", v)
}

func (t *Transformer) getStringSlice(column string) []string {
	v, ok := t.Payload[column]
	if !ok || v == nil {
		return nil
	}

	t.Visited[column] = true

	var s string

	switch val := v.(type) {
	case []string:
		return val
	case []uint8:
		s = string(val)
		if s == "" {
			return nil
		}
		return strings.Split(s, ",")
	case string:
		if val == "" {
			return nil
		}
		return strings.Split(val, ",")
	default:
		s := fmt.Sprintf("%v", v)
		if s == "" {
			return nil
		}
		return strings.Split(s, ",")
	}
}

func (t *Transformer) getTime(column string) *time.Time {
	v, ok := t.Payload[column]
	if !ok || v == nil {
		return nil
	}

	var val time.Time
	var err error

	switch v := v.(type) {
	case time.Time:
		val = v
	case []uint8:
		s := string(v)
		val, err = parseTime(s)
		if err != nil {
			_ = log.Warnf("error parsing time string '%s' for column '%s': %v", s, column, err)
			return nil
		}
	case string:
		val, err = parseTime(v)
		if err != nil {
			_ = log.Warnf("error parsing time string '%s' for column '%s': %v", v, column, err)
			return nil
		}
	default:
		_ = log.Warnf("unsupported type for time conversion: %T for column '%s'", v, column)
		return nil
	}
	t.Visited[column] = true
	return &val
}

func parseTime(s string) (time.Time, error) {
	layouts := []string{
		time.DateTime,
		time.DateOnly,
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999", // Common format for DATETIME/TIMESTAMP with fractional seconds
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not parse '%s' with any known layouts", s)
}

func (t *Transformer) getInt(column string) int {
	v, ok := t.Payload[column]
	if !ok || v == nil {
		return 0
	}

	t.Visited[column] = true

	switch val := v.(type) {
	case int:
		return val
	case int8:
		return int(val)
	case int16:
		return int(val)
	case int32:
		return int(val)
	case int64:
		return int(val)
	case float32:
		return int(val)
	case float64:
		return int(val)
	case []uint8:
		s := string(val)
		if i, err := strconv.Atoi(s); err == nil {
			return i
		} else {
			_ = log.Warnf("error parsing int string '%s' for column '%s': %v", s, column, err)
			t.Visited[column] = false
			return 0
		}
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		} else {
			_ = log.Warnf("error parsing int string '%s' for column '%s': %v", val, column, err)
			t.Visited[column] = false
			return 0
		}
	default:
		s := fmt.Sprintf("%v", v)
		if i, err := strconv.Atoi(s); err == nil {
			return i
		} else {
			_ = log.Warnf("error parsing int from unsupported type %T for column '%s': %v", v, column, err)
			t.Visited[column] = false
			return 0
		}
	}
}

func (t *Transformer) getRaw(column string) interface{} {
	if v, ok := t.Payload[column]; ok {
		t.Visited[column] = true
		return v
	}
	return nil
}

func isUserEmpty(u *core.UserInfo) bool {
	if u == nil {
		return true
	}
	return u.UserAvatar == "" && u.UserID == "" && u.UserName == ""
}
