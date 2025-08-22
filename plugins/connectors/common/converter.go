/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"infini.sh/coco/modules/common"
)

type Transformer struct {
	Payload map[string]interface{}
	Visited map[string]bool
}

func (t *Transformer) Transform(doc *common.Document, m *Mapping) {
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

	owner := &common.UserInfo{
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

	doc.LastUpdatedBy = &common.EditorInfo{
		UpdatedAt: t.getTime(m.LastUpdatedBy.Timestamp),
	}

	user := &common.UserInfo{
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
	if v, ok := t.Payload[column]; ok {
		t.Visited[column] = true
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func (t *Transformer) getStringSlice(column string) []string {
	if v, ok := t.Payload[column]; ok {
		if val, ok := v.(string); ok {
			t.Visited[column] = true
			return strings.Split(val, ",")
		}
	}
	return nil
}

func (t *Transformer) getTime(column string) *time.Time {
	if v, ok := t.Payload[column]; ok {
		if val, ok := v.(time.Time); ok {
			t.Visited[column] = true
			return &val
		}
	}
	return nil
}

func (t *Transformer) getInt(column string) int {
	if val, ok := t.Payload[column]; ok {
		s := fmt.Sprintf("%v", val)
		if v, err := strconv.Atoi(s); err == nil {
			t.Visited[column] = true
			return v
		}
	}
	return 0
}

func (t *Transformer) getRaw(column string) interface{} {
	if v, ok := t.Payload[column]; ok {
		t.Visited[column] = true
		return v
	}
	return nil
}

func isUserEmpty(u *common.UserInfo) bool {
	if u == nil {
		return true
	}
	return u.UserAvatar == "" && u.UserID == "" && u.UserName == ""
}
