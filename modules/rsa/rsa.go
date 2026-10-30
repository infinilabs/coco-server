/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

// Package rsa implements the Remote Source Adapter leg (DESIGN-RSA.md):
// query-time federation over external sources — ES-compatible clusters
// first, SQL databases later — with per-source timeout / retry×failover /
// QPS / concurrency / cache budgets. Hits map straight into
// core.Document; nothing is ever copied into the local index.
package rsa

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
)

// Query is the adapter-agnostic retrieval request. From/Size carry the
// page window already clamped by the source's MaxWindow.
type Query struct {
	Text string
	From int
	Size int
	// Extra is reserved for filter push-down (P2+); adapters ignore it.
	Extra map[string]interface{}
}

// MappedHit is one row already mapped onto the standard document shape.
// Rank is the source's own order (1-based) — RRF consumes ranks, never
// native scores, so heterogeneous sources need no normalization.
type MappedHit struct {
	Document core.Document
	Rank     int
}

// Result is the adapter's normalized answer.
type Result struct {
	Total  int64
	Hits   []MappedHit
	TookMS int64
}

// Attempt records one try against one endpoint — the debug/诊断 grain.
type Attempt struct {
	Endpoint string `json:"endpoint"`
	Err      string `json:"error,omitempty"`
	TookMS   int64  `json:"took_ms"`
	Final    bool   `json:"final"` // true on the attempt that produced the result
}

// SourceDiag is the per-source slice of leg diagnostics (search log and
// the _debug endpoint both render this).
type SourceDiag struct {
	SourceID   string    `json:"source_id"`
	SourceName string    `json:"source_name"`
	Adapter    string    `json:"adapter"`
	Status     string    `json:"status"` // ok | timeout | error | skipped | cache
	Hits       int       `json:"hits"`
	TookMS     int64     `json:"took_ms"`
	CacheHit   bool      `json:"cache_hit"`
	Attempts   []Attempt `json:"attempts,omitempty"`
	Err        string    `json:"error,omitempty"`
}

// Adapter translates one Query into a source-native search and executes
// it against ONE endpoint (the executor owns endpoint rotation, retries
// and the shared deadline — adapters stay dumb transport+mapping).
type Adapter interface {
	Type() string
	// Validate checks connection/query/mapping shape at save time.
	Validate(src *core.RemoteSource) error
	// Search runs against the given endpoint only. The ctx already
	// carries the per-attempt deadline from the executor.
	Search(ctx context.Context, src *core.RemoteSource, endpoint string, q Query) (Result, error)
}

var (
	adapterMu sync.RWMutex
	adapters  = map[string]Adapter{}
)

// Register installs a built-in adapter; init() order is irrelevant.
func Register(a Adapter) {
	adapterMu.Lock()
	defer adapterMu.Unlock()
	adapters[a.Type()] = a
}

// GetAdapter resolves the adapter implementation by type.
func GetAdapter(typ string) (Adapter, error) {
	adapterMu.RLock()
	defer adapterMu.RUnlock()
	a, ok := adapters[typ]
	if !ok {
		return nil, fmt.Errorf("unknown rsa adapter [%s]", typ)
	}
	return a, nil
}

// AdapterTypes lists registered adapter types (for the UI select).
func AdapterTypes() []string {
	adapterMu.RLock()
	defer adapterMu.RUnlock()
	out := make([]string, 0, len(adapters))
	for k := range adapters {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ResolveEnabled returns the enabled sources, priority-desc, capped at
// max. Ordering ties break by name for stable fan-out rotation.
func ResolveEnabled(ctx context.Context, max int) ([]core.RemoteSource, error) {
	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	octx.PermissionScope(security.PermissionScopePlatform)
	orm.WithModel(octx, &core.RemoteSource{})

	q := orm.NewQuery().Size(1000).Filter(orm.TermQuery("enabled", true))
	res, err := orm.SearchV2(octx, q)
	if err != nil {
		return nil, err
	}
	sources, _, err := elastic.DecodeHits[core.RemoteSource](res)
	if err != nil {
		return nil, err
	}
	sort.Slice(sources, func(i, j int) bool {
		if sources[i].Priority != sources[j].Priority {
			return sources[i].Priority > sources[j].Priority
		}
		return sources[i].Name < sources[j].Name
	})
	if max > 0 && len(sources) > max {
		sources = sources[:max]
	}
	return sources, nil
}

// GetSource loads one source by id (management/test/debug paths).
func GetSource(ctx context.Context, id string) (core.RemoteSource, bool, error) {
	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	octx.PermissionScope(security.PermissionScopePlatform)
	orm.WithModel(octx, &core.RemoteSource{})

	src := core.RemoteSource{}
	src.ID = id
	exists, err := orm.GetV2(octx, &src)
	return src, exists, err
}

// MaskSource zeroes credential fields for API reads; the admin UI shows
// a placeholder instead of secrets.
func MaskSource(src *core.RemoteSource) core.RemoteSource {
	out := *src
	if out.Connection.Password != "" {
		out.Connection.Password = "******"
	}
	if out.Connection.APIKey != "" {
		out.Connection.APIKey = "******"
	}
	return out
}

// MaskedPlaceholder marks which credential fields were masked, so the UI
// can keep "******" as "unchanged" on save.
const MaskedCredential = "******"

// CleanMaskedCredentials keeps stored secrets when the client round-trips
// the masked value back unchanged (the standard edit flow).
func CleanMaskedCredentials(incoming, stored *core.RemoteSource) {
	if incoming.Connection.Password == MaskedCredential {
		incoming.Connection.Password = stored.Connection.Password
	}
	if incoming.Connection.APIKey == MaskedCredential {
		incoming.Connection.APIKey = stored.Connection.APIKey
	}
}
