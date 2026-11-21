/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rsa

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/rate"

	xrate "golang.org/x/time/rate"
)

const (
	defaultTimeoutMS     = 3000
	timeoutHardCapMS     = 10000
	defaultMaxConcurrent = 2
	defaultQPS           = 2
	defaultMaxAttempts   = 2
	maxAttemptsHardCap   = 3
	defaultMaxWindow     = 100
	maxWindowHardCap     = 1000
	defaultBackoffMS     = 150
)

// errSourceTimeout marks a budget exhaustion — classified "timeout", not
// "error", and by default not retryable (the budget is already spent).
var errSourceTimeout = errors.New("rsa: source budget exhausted")

// effectiveSettings is nil-safe global config access for defaults. A
// missing config store (fresh dev binary, unit tests) degrades to the
// zero settings — every default then applies; the leg itself is gated by
// the caller reading the same switch.
func effectiveSettings() (s core.SearchSettings) {
	defer func() {
		if r := recover(); r != nil {
			s = core.SearchSettings{}
		}
	}()
	cfg := common.AppConfig()
	if cfg.SearchSettings != nil {
		return *cfg.SearchSettings
	}
	return core.SearchSettings{}
}

// endpointCursor is the per-source sticky rotation offset: every attempt
// starts at the next endpoint, so a dead node is naturally skipped after
// one failure and healthy traffic spreads across the ring.
var (
	cursorMu    sync.Mutex
	endpointCur = map[string]int{}
)

func endpointCount(src *core.RemoteSource) int {
	return len(src.Connection.Endpoints)
}

// beginRotation advances the sticky cursor once per QUERY and returns the
// rotation base; attempts then walk the ring from that base. Advancing
// inside nextEndpoint-per-attempt double-counted the rotation (attempt 1
// re-read the already-advanced cursor and landed back on the failed
// endpoint with 2-endpoint rings).
func beginRotation(src *core.RemoteSource) int {
	n := endpointCount(src)
	if n == 0 {
		return 0
	}
	cursorMu.Lock()
	defer cursorMu.Unlock()
	cur := endpointCur[src.ID]
	endpointCur[src.ID] = (cur + 1) % n
	return cur
}

// endpointAt resolves the endpoint for attempt k from a rotation base.
func endpointAt(src *core.RemoteSource, base, attempt int) string {
	n := endpointCount(src)
	if n == 0 {
		return ""
	}
	return src.Connection.Endpoints[(base+attempt)%n]
}

// sourceSemaphores bounds in-flight queries per source.
var (
	semMu    sync.Mutex
	semaphores = map[string]chan struct{}{}
)

func semaphoreFor(src *core.RemoteSource) chan struct{} {
	n := src.Controls.MaxConcurrent
	if n <= 0 {
		n = defaultMaxConcurrent
	}
	if n > 16 {
		n = 16
	}
	semMu.Lock()
	defer semMu.Unlock()
	if c, ok := semaphores[src.ID]; ok {
		return c
	}
	c := make(chan struct{}, n)
	semaphores[src.ID] = c
	return c
}

func limiterFor(src *core.RemoteSource) *xrate.Limiter {
	qps := src.Controls.QPS
	if qps == 0 {
		qps = defaultQPS
	}
	if qps < 0 {
		return nil // explicit unlimited
	}
	return rate.GetRateLimiterPerSecond("rsa", src.ID, qps)
}

func budgetFor(src *core.RemoteSource) time.Duration {
	ms := src.Controls.TimeoutMS
	if ms <= 0 {
		ms = defaultTimeoutMS
	}
	if ms > timeoutHardCapMS {
		ms = timeoutHardCapMS
	}
	return time.Duration(ms) * time.Millisecond
}

// ClampWindow applies from+size against the source's MaxWindow.
func ClampWindow(src *core.RemoteSource, from, size int) (int, int) {
	max := src.Controls.MaxWindow
	if max <= 0 {
		max = defaultMaxWindow
	}
	if max > maxWindowHardCap {
		max = maxWindowHardCap
	}
	if from < 0 {
		from = 0
	}
	if size <= 0 {
		size = 10
	}
	if from+size > max {
		over := from + size - max
		if over >= size {
			return from, 0 // window entirely beyond the cap — no rows
		}
		size -= over
	}
	return from, size
}

// retryableClass classifies an adapter error for the retry policy:
// "", "timeout", or "error" plus whether a retry may help. Only
// connection-class failures and 5xx earn a retry by default.
func retryableClass(err error) (class string, retriable bool) {
	if err == nil {
		return "", false
	}
	if errors.Is(err, errSourceTimeout) || errors.Is(err, context.DeadlineExceeded) {
		return "timeout", false
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "no such host"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "broken pipe"),
		strings.Contains(msg, "i/o timeout"),
		strings.Contains(msg, "tls handshake timeout"),
		strings.Contains(msg, "eof"):
		return "conn", true
	case strings.Contains(msg, "status 5"), strings.Contains(msg, "502"), strings.Contains(msg, "503"), strings.Contains(msg, "504"):
		return "5xx", true
	}
	return "error", false
}

// ExecuteQuery runs one query against one source under its full
// protection envelope: cache → QPS limiter → concurrency semaphore →
// single deadline shared by every attempt → endpoint rotation with
// bounded retries. The SourceDiag carries the attempt ring for search
// logs and the debug endpoint.
func ExecuteQuery(parent context.Context, src *core.RemoteSource, q Query) (*Result, *SourceDiag) {
	diag := &SourceDiag{
		SourceID:   src.ID,
		SourceName: src.Name,
		Adapter:    src.Adapter,
	}
	stats := statsFor(src.ID)
	stats.Requests.Add(1)

	if r := cacheGet(src, q); r != nil {
		stats.CacheHits.Add(1)
		diag.Status = "cache"
		diag.CacheHit = true
		diag.Hits = len(r.Hits)
		return r, diag
	}

	adapter, err := GetAdapter(src.Adapter)
	if err != nil {
		stats.Errors.Add(1)
		stats.recordError(err.Error())
		diag.Status = "error"
		diag.Err = err.Error()
		return nil, diag
	}

	// one deadline for limiter wait + every attempt
	ctx, cancel := context.WithTimeout(parent, budgetFor(src))
	defer cancel()

	if l := limiterFor(src); l != nil {
		if err := l.Wait(ctx); err != nil {
			stats.Timeouts.Add(1)
			stats.recordError("rate limiter: budget exhausted while waiting")
			diag.Status = "timeout"
			diag.Err = "blocked by source QPS limit before dispatch"
			return nil, diag
		}
	}

	select {
	case semaphoreFor(src) <- struct{}{}:
		defer func() { <-semaphoreFor(src) }()
	case <-ctx.Done():
		stats.Timeouts.Add(1)
		stats.recordError("concurrency slot: budget exhausted while queued")
		diag.Status = "timeout"
		diag.Err = "blocked by source concurrency limit before dispatch"
		return nil, diag
	}

	maxAttempts := src.Controls.RetryMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}
	if maxAttempts > maxAttemptsHardCap {
		maxAttempts = maxAttemptsHardCap
	}
	backoff := time.Duration(defaultBackoffMS) * time.Millisecond
	if src.Controls.RetryBackoffMS > 0 {
		backoff = time.Duration(src.Controls.RetryBackoffMS) * time.Millisecond
	}

	start := time.Now()
	base := beginRotation(src)
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// spend the pause only when the remaining budget fits it
			deadline, _ := ctx.Deadline()
			remain := time.Until(deadline)
			if remain < backoff+50*time.Millisecond {
				break
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
			}
		}

		endpoint := endpointAt(src, base, attempt)
		aStart := time.Now()
		// per-attempt context: the shared deadline minus nothing extra
		res, aerr := adapter.Search(ctx, src, endpoint, q)
		attemptRow := Attempt{Endpoint: endpoint, TookMS: time.Since(aStart).Milliseconds()}
		stats.Attempts.Add(1)

		if aerr == nil {
			attemptRow.Final = true
			diag.Attempts = append(diag.Attempts, attemptRow)
			elapsed := time.Since(start)
			diag.Status = "ok"
			diag.TookMS = elapsed.Milliseconds()
			diag.Hits = len(res.Hits)
			res.TookMS = diag.TookMS
			stats.Hits.Add(int64(len(res.Hits)))
			stats.recordLatency(diag.TookMS)
			cachePut(src, q, res)
			return &res, diag
		}

		attemptRow.Err = aerr.Error()
		diag.Attempts = append(diag.Attempts, attemptRow)
		lastErr = aerr
		class, retriable := retryableClass(aerr)
		if attempt > 0 {
			stats.Failovers.Add(1) // this attempt ran on a rotated endpoint
		}
		if ctx.Err() != nil {
			break // budget spent — further attempts are pointless
		}
		if !retriable && !(class == "timeout" && src.Controls.RetryOnTimeout) {
			break
		}
	}

	elapsed := time.Since(start)
	class, _ := retryableClass(lastErr)
	if class == "timeout" || ctx.Err() != nil {
		stats.Timeouts.Add(1)
		diag.Status = "timeout"
	} else {
		stats.Errors.Add(1)
		diag.Status = "error"
	}
	if lastErr != nil {
		diag.Err = lastErr.Error()
		stats.recordError(diag.Err)
	}
	diag.TookMS = elapsed.Milliseconds()
	return nil, diag
}

// httpStatusError lets adapters surface transport status codes in a
// retry-classifiable way.
type httpStatusError struct {
	Code int
	Body string
}

func (e *httpStatusError) Error() string {
	body := e.Body
	if len(body) > 200 {
		body = body[:200]
	}
	return fmt.Sprintf("rsa: http status %d: %s", e.Code, body)
}

var _ error = (*httpStatusError)(nil)
