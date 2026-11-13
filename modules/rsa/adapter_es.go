/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rsa

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

// AdapterEsCompatible speaks the Elasticsearch _search family — one
// transport for ES, Easysearch and OpenSearch (same DSL origin; only
// score details differ, and RRF ignores native scores anyway).
type AdapterEsCompatible struct{}

const AdapterTypeEsCompatible = "es_compatible"

func init() {
	Register(&AdapterEsCompatible{})
}

func (a *AdapterEsCompatible) Type() string { return AdapterTypeEsCompatible }

func (a *AdapterEsCompatible) Validate(src *core.RemoteSource) error {
	if len(src.Connection.Endpoints) == 0 {
		return fmt.Errorf("es_compatible: at least one endpoint is required")
	}
	for _, e := range src.Connection.Endpoints {
		if !strings.HasPrefix(e, "http://") && !strings.HasPrefix(e, "https://") {
			return fmt.Errorf("es_compatible: endpoint [%s] must start with http:// or https://", e)
		}
	}
	if strings.TrimSpace(src.Query.Indices) == "" {
		return fmt.Errorf("es_compatible: indices pattern is required")
	}
	switch src.Connection.Auth {
	case "", "basic", "api_key":
	default:
		return fmt.Errorf("es_compatible: auth must be one of basic|api_key (empty = none), got [%s]", src.Connection.Auth)
	}
	if tpl := strings.TrimSpace(src.Query.QueryTemplate); tpl != "" {
		var probe map[string]interface{}
		if err := util.FromJSONBytes([]byte(tpl), &probe); err != nil {
			return fmt.Errorf("es_compatible: query_template must be a JSON object: %w", err)
		}
	}
	return nil
}

// buildBody renders the _search request body: the operator's template
// ({{.Text}}/{{.From}}/{{.Size}} placeholder substitution — no template
// engine, the values are plain integers and an escaped string) or the
// built-in multi_match default.
func (a *AdapterEsCompatible) buildBody(src *core.RemoteSource, q Query) (map[string]interface{}, error) {
	if tpl := strings.TrimSpace(src.Query.QueryTemplate); tpl != "" {
		rendered := strings.ReplaceAll(tpl, "{{.Text}}", jsonEscapeString(q.Text))
		rendered = strings.ReplaceAll(rendered, "{{.From}}", fmt.Sprintf("%d", q.From))
		rendered = strings.ReplaceAll(rendered, "{{.Size}}", fmt.Sprintf("%d", q.Size))
		var body map[string]interface{}
		if err := util.FromJSONBytes([]byte(rendered), &body); err != nil {
			return nil, fmt.Errorf("query_template rendered invalid JSON: %w", err)
		}
		return body, nil
	}
	// default: multi_match across common text fields, deterministic sort
	// (page stability requires a total order, DESIGN-RSA.md §9)
	return map[string]interface{}{
		"from": q.From,
		"size": q.Size,
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":     q.Text,
				"fields":    []string{"title^3", "name^2", "tags^2", "content", "summary", "message"},
				"type":      "best_fields",
				"operator":  "or",
			},
		},
		"sort": []interface{}{
			map[string]interface{}{"_score": "desc"},
			map[string]interface{}{"_id": "asc"},
		},
	}, nil
}

func jsonEscapeString(s string) string {
	b, _ := json.Marshal(s)
	return strings.Trim(string(b), `"`)
}

func (a *AdapterEsCompatible) httpClient(src *core.RemoteSource) *http.Client {
	transport := &http.Transport{
		MaxIdleConnsPerHost: 4,
	}
	if !src.Connection.VerifyTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{Transport: transport}
}

func (a *AdapterEsCompatible) applyAuth(req *http.Request, src *core.RemoteSource) {
	switch src.Connection.Auth {
	case "basic":
		req.SetBasicAuth(src.Connection.Username, src.Connection.Password)
	case "api_key":
		req.Header.Set("X-API-Key", src.Connection.APIKey)
	}
}

// Search executes one attempt against one endpoint. Transport errors and
// non-2xx statuses come back as errors the executor can classify
// (connection class / 5xx → retry+failover).
func (a *AdapterEsCompatible) Search(ctx context.Context, src *core.RemoteSource, endpoint string, q Query) (Result, error) {
	if endpoint == "" {
		return Result{}, fmt.Errorf("es_compatible: no endpoint configured")
	}
	body, err := a.buildBody(src, q)
	if err != nil {
		return Result{}, err
	}
	payload, err := util.ToJSONBytes(body)
	if err != nil {
		return Result{}, err
	}

	url := strings.TrimRight(endpoint, "/") + "/" + strings.TrimSpace(src.Query.Indices) + "/_search"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := a.httpClient(src).Do(req)
	if err != nil {
		return Result{}, err // transport error — executor classifies
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if resp.StatusCode >= 400 {
		return Result{}, &httpStatusError{Code: resp.StatusCode, Body: string(raw)}
	}

	var esResp struct {
		Took  int64 `json:"took"`
		Hits  struct {
			Total map[string]interface{} `json:"total"`
			Hits  []struct {
				ID      string                 `json:"_id"`
				Index   string                 `json:"_index"`
				Score   float64                `json:"_score"`
				Source  map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := util.FromJSONBytes(raw, &esResp); err != nil {
		return Result{}, fmt.Errorf("es_compatible: invalid _search response: %w", err)
	}

	out := Result{TookMS: time.Since(start).Milliseconds()}
	if v, ok := esResp.Hits.Total["value"].(float64); ok {
		out.Total = int64(v)
	}
	for i, h := range esResp.Hits.Hits {
		fields := map[string]interface{}{}
		for k, v := range h.Source {
			fields[k] = v
		}
		// flattened access keys — mapping paths may address _id/_index/_score
		fields["_id"] = h.ID
		fields["_index"] = h.Index
		fields["_score"] = h.Score
		out.Hits = append(out.Hits, MapHit(src, RawHit{ID: h.ID, Index: h.Index, Fields: fields}, i+1))
	}
	return out, nil
}
