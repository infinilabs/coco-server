/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rsa

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

// RawHit is the adapter-normalized row handed to the mapper: a flat map
// (ES: _id/_index/_score/_source.* flattened; SQL later: column→value).
// Field paths in the mapping resolve with dot notation against it.
type RawHit struct {
	ID    string
	Index string
	Fields map[string]interface{}
}

// lookupPath resolves a dot path ("" returns nil). Map descent only —
// arrays are not traversed in P1.
func lookupPath(m map[string]interface{}, path string) interface{} {
	if path == "" || m == nil {
		return nil
	}
	cur := m
	parts := strings.Split(path, ".")
	for i, p := range parts {
		v, ok := cur[p]
		if !ok {
			return nil
		}
		if i == len(parts)-1 {
			return v
		}
		next, ok := v.(map[string]interface{})
		if !ok {
			return nil
		}
		cur = next
	}
	return nil
}

func fieldString(m map[string]interface{}, path, fallbackPath string) string {
	if v := lookupPath(m, path); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	if v := lookupPath(m, fallbackPath); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// PseudoID projects a raw source id into a deterministic document id —
// pagination stability and dedup both lean on it (DESIGN-RSA.md §4.2).
func PseudoID(sourceID, rawID string) string {
	h := sha1.Sum([]byte(rawID))
	return "rsa_" + sourceID + "_" + hex.EncodeToString(h[:8])
}

// MapHit renders one raw hit as a standard document. Mapping misses
// leave fields empty and are recorded in the debug notes — a row is
// never dropped over mapping (G2).
func MapHit(src *core.RemoteSource, raw RawHit, rank int) MappedHit {
	fields := raw.Fields
	if fields == nil {
		fields = map[string]interface{}{}
	}

	titlePath := src.Mapping.Title
	contentPath := src.Mapping.Content
	urlPath := src.Mapping.URL
	tagsPath := src.Mapping.Tags

	doc := core.Document{}
	doc.ID = PseudoID(src.ID, raw.ID)
	doc.Title = fieldString(fields, titlePath, "title")
	doc.Content = fieldString(fields, contentPath, "content")
	doc.URL = fieldString(fields, urlPath, "url")

	created := fieldString(fields, src.Mapping.Created, "created")
	if created != "" {
		if t, err := time.Parse(time.RFC3339, created); err == nil {
			doc.Created = &t
		} else {
			now := time.Now()
			doc.Created = &now
		}
	} else {
		now := time.Now()
		doc.Created = &now
	}

	if tagsPath == "" {
		tagsPath = "tags"
	}
	if v := lookupPath(fields, tagsPath); v != nil {
		switch t := v.(type) {
		case []interface{}:
			for _, e := range t {
				doc.Tags = append(doc.Tags, fmt.Sprintf("%v", e))
			}
		case string:
			if t != "" {
				doc.Tags = []string{t}
			}
		}
	}

	doc.Type = "rsa_document"
	doc.Source = core.DataSourceReference{
		ID:   "rsa_" + src.ID,
		Name: src.Name,
		Type: "rsa",
		Icon: "cloud_server",
	}
	doc.Status = core.DocumentStatusCompleted
	doc.Processed = true
	doc.Metadata = util.MapStr{
		"rsa": util.MapStr{
			"source_id":   src.ID,
			"source_name": src.Name,
			"adapter":     src.Adapter,
			"raw_id":      raw.ID,
			"index":       raw.Index,
			"rank":        rank,
		},
	}
	return MappedHit{Document: doc, Rank: rank}
}
