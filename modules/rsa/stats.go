/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rsa

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// SourceStats is the per-source counters behind GET /rsa/_stats —
// in-memory atomics, never persisted in P1 (the search log carries the
// per-query diagnostics). The latency ring feeds P50/P95 without a
// histogram library.
type SourceStats struct {
	Requests  atomic.Int64
	Hits      atomic.Int64
	Timeouts  atomic.Int64
	Errors    atomic.Int64
	CacheHits atomic.Int64
	Attempts  atomic.Int64 // includes retries/failovers
	Failovers atomic.Int64 // attempts that ran on a different endpoint than the previous one

	mu      sync.Mutex
	latency []int64 // ring, capped
	lastErr atomic.Value
	lastAt  atomic.Int64 // unix ms
}

const latencyRingSize = 128

var (
	statsMu    sync.RWMutex
	statsStore = map[string]*SourceStats{}
)

func statsFor(sourceID string) *SourceStats {
	statsMu.RLock()
	s, ok := statsStore[sourceID]
	statsMu.RUnlock()
	if ok {
		return s
	}
	statsMu.Lock()
	defer statsMu.Unlock()
	if s, ok := statsStore[sourceID]; ok {
		return s
	}
	s = &SourceStats{}
	statsStore[sourceID] = s
	return s
}

func (s *SourceStats) recordLatency(ms int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.latency) >= latencyRingSize {
		s.latency = s.latency[1:]
	}
	s.latency = append(s.latency, ms)
}

func (s *SourceStats) recordError(err string) {
	s.lastErr.Store(err)
	s.lastAt.Store(time.Now().UnixMilli())
}

// percentiles returns p50/p95/max of the latency ring (0 when empty).
func (s *SourceStats) percentiles() (p50, p95, max int64) {
	s.mu.Lock()
	buf := make([]int64, len(s.latency))
	copy(buf, s.latency)
	s.mu.Unlock()
	if len(buf) == 0 {
		return 0, 0, 0
	}
	sort.Slice(buf, func(i, j int) bool { return buf[i] < buf[j] })
	pick := func(p float64) int64 {
		idx := int(float64(len(buf)-1) * p)
		return buf[idx]
	}
	return pick(0.5), pick(0.95), buf[len(buf)-1]
}

// StatsSnapshot is the JSON shape of GET /rsa/_stats.
type StatsSnapshot struct {
	SourceID   string `json:"source_id"`
	Requests   int64  `json:"requests"`
	Hits       int64  `json:"hits"`
	Timeouts   int64  `json:"timeouts"`
	Errors     int64  `json:"errors"`
	CacheHits  int64  `json:"cache_hits"`
	Attempts   int64  `json:"attempts"`
	Failovers  int64  `json:"failovers"`
	LatencyP50 int64  `json:"latency_p50_ms"`
	LatencyP95 int64  `json:"latency_p95_ms"`
	LatencyMax int64  `json:"latency_max_ms"`
	LastError  string `json:"last_error,omitempty"`
	LastAt     string `json:"last_at,omitempty"` // RFC3339
}

// SnapshotStats renders live counters for every source ever queried.
func SnapshotStats() []StatsSnapshot {
	statsMu.RLock()
	ids := make([]string, 0, len(statsStore))
	for id := range statsStore {
		ids = append(ids, id)
	}
	rows := make([]*SourceStats, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, statsStore[id])
	}
	statsMu.RUnlock()

	sort.Strings(ids)
	out := make([]StatsSnapshot, 0, len(ids))
	for i, id := range ids {
		s := rows[i]
		p50, p95, mx := s.percentiles()
		snap := StatsSnapshot{
			SourceID:   id,
			Requests:   s.Requests.Load(),
			Hits:       s.Hits.Load(),
			Timeouts:   s.Timeouts.Load(),
			Errors:     s.Errors.Load(),
			CacheHits:  s.CacheHits.Load(),
			Attempts:   s.Attempts.Load(),
			Failovers:  s.Failovers.Load(),
			LatencyP50: p50,
			LatencyP95: p95,
			LatencyMax: mx,
		}
		if v, ok := s.lastErr.Load().(string); ok {
			snap.LastError = v
		}
		if at := s.lastAt.Load(); at > 0 {
			snap.LastAt = time.UnixMilli(at).UTC().Format(time.RFC3339)
		}
		out = append(out, snap)
	}
	return out
}
