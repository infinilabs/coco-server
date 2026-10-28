/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"infini.sh/framework/core/orm"
)

// RemoteSource (RSA — Remote Source Adapter): one externally queryable
// source that participates in hybrid search as a federated leg. The
// query-time complement of the sync connectors: nothing is copied into
// the local index — hits are mapped straight into core.Document rows and
// fused via RRF. Low-frequency by design; every source carries its own
// timeout / concurrency / QPS / cache budget so the leg can never take a
// backend down (DESIGN-RSA.md §5).
type RemoteSource struct {
	orm.ORMObjectBase

	Name        string `json:"name" elastic_mapping:"name:{type:keyword,fields:{text: {type: text}}}"`
	Description string `json:"description,omitempty" elastic_mapping:"description:{type:text}"`

	// Adapter selects the built-in adapter implementation, e.g.
	// "es_compatible". The registry lives in modules/rsa.
	Adapter string `json:"adapter" elastic_mapping:"adapter:{type:keyword}"`

	// Enabled gates the leg: disabled sources are never resolved for
	// search (management APIs still see them).
	Enabled bool `json:"enabled" elastic_mapping:"enabled:{type:boolean}"`

	// Priority breaks the fan-out tie when more sources are enabled than
	// search_settings.rsa_max_sources_per_query allows — higher first.
	Priority int `json:"priority,omitempty" elastic_mapping:"priority:{type:integer}"`

	Connection RSAConnection    `json:"connection" elastic_mapping:"connection:{enabled:false}"`
	Query      RSAQuery         `json:"query" elastic_mapping:"query:{enabled:false}"`
	Mapping    RSADocumentMapping `json:"mapping" elastic_mapping:"mapping:{enabled:false}"`
	Controls   RSAControls      `json:"controls" elastic_mapping:"controls:{enabled:false}"`
}

// RSAConnection is adapter-agnostic addressing plus auth. Credentials are
// admin-only fields on an admin-only resource; routes register the
// sensitive-field mask feature and reads mask them.
type RSAConnection struct {
	Endpoints []string `json:"endpoints"` // failover ring, rotated per attempt
	// Auth: "" (none) | "basic" | "api_key" (X-API-Key header)
	Auth     string `json:"auth,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
	// VerifyTLS defaults to true; self-signed labs may disable per source.
	VerifyTLS bool `json:"verify_tls"`
}

// RSAQuery carries the adapter's source-side query definition. The
// es_compatible adapter reads Indices + QueryTemplate.
type RSAQuery struct {
	// Indices is the ES index pattern to search, e.g. "logs-*,wiki*".
	Indices string `json:"indices,omitempty"`
	// QueryTemplate: optional raw JSON DSL body with {{.Text}} {{.From}}
	// {{.Size}} placeholders; empty = built-in multi_match template.
	QueryTemplate string `json:"query_template,omitempty"`
}

// RSADocumentMapping maps a raw hit (map form, adapter-normalized: SQL
// columns or ES _source) onto the standard document fields. Values are
// dot paths into the raw hit; empty falls back to adapter defaults.
// Missing paths leave the field empty — a row is never dropped over a
// mapping miss.
type RSADocumentMapping struct {
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
	URL     string `json:"url,omitempty"`
	Created string `json:"created,omitempty"`
	Tags    string `json:"tags,omitempty"`
	// IDPath overrides the raw-id path (default: the adapter's native id
	// field, e.g. _id / _row._id); the projection into a deterministic
	// pseudo document id always applies on top.
	IDPath string `json:"id_path,omitempty"`
}

// RSAControls is the per-source protection budget (DESIGN-RSA.md §5):
// timeout, retry×failover sharing ONE deadline, QPS, concurrency, cache.
type RSAControls struct {
	// TimeoutMS is the total budget for ALL attempts of one query,
	// including retries and endpoint failover. Default 3000, hard cap 10000.
	TimeoutMS int `json:"timeout_ms,omitempty"`
	// MaxConcurrent bounds in-flight queries per source (default 2).
	MaxConcurrent int `json:"max_concurrent,omitempty"`
	// QPS caps the per-source rate (default 2, 0 = capped by config
	// default, -1 = unlimited). RSA is low-frequency by contract.
	QPS int `json:"qps,omitempty"`
	// CacheTTLSeconds: per-source result cache TTL (default 60, 0 = off).
	CacheTTLSeconds int `json:"cache_ttl_seconds,omitempty"`
	// MaxWindow caps from+size per query (default 100, hard cap 1000).
	MaxWindow int `json:"max_window,omitempty"`

	// Retry: bounded attempt ring within the SAME deadline.
	// MaxAttempts includes the first try (default 2, hard cap 3).
	RetryMaxAttempts int `json:"retry_max_attempts,omitempty"`
	// RetryBackoffMS is the fixed pause between attempts (default 150);
	// skipped when the remaining budget cannot fit it.
	RetryBackoffMS int `json:"retry_backoff_ms,omitempty"`
	// RetryOnTimeout opts into retrying timed-out attempts. Off by
	// default: a timeout means the budget is spent — retrying only
	// amplifies latency (DESIGN-RSA.md §5.4).
	RetryOnTimeout bool `json:"retry_on_timeout,omitempty"`
}
