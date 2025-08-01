// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package filter

import (
	"crypto/sha256"
	"encoding/hex"
	"infini.sh/framework/core/global"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
)

func init() {
	f := &FingerprintThrottleFilter{}

	// Register cleanup task only once
	global.RegisterBackgroundCallback(&global.BackgroundTask{
		Tag:      "fingerprint_throttle_filter_cleanup",
		Func:     func() { f.cleanupOldEntries() },
		Interval: 10 * time.Second,
	})

	api.RegisterUIFilter(f)
}

type FingerprintThrottleFilter struct {
	api.Handler
	recent sync.Map
}

func (f *FingerprintThrottleFilter) GetPriority() int {
	return 20
}

const FeatureFingerprintThrottle = "fingerprint_throttle"
const throttleWindow = 100 * time.Millisecond

func (f *FingerprintThrottleFilter) ApplyFilter(
	method string,
	pattern string,
	options *api.HandlerOptions,
	next httprouter.Handle,
) httprouter.Handle {

	//option not enabled
	if options == nil || !options.Feature(FeatureFingerprintThrottle) {
		log.Debug(method, ",", pattern, ",skip feature ", FeatureFingerprintThrottle)
		return next
	}

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		fingerprint, err := f.computeFingerprint(r)
		if global.Env().IsDebug {
			log.Tracef("req: %v, fingerprint: %v", r.URL.String(), fingerprint)
		}
		if err != nil {
			log.Warn("could not compute fingerprint:", err)
			next(w, r, ps)
			return
		}

		now := time.Now()

		if tsRaw, exists := f.recent.Load(fingerprint); exists {
			if ts, ok := tsRaw.(time.Time); ok && now.Sub(ts) < throttleWindow {
				log.Warnf("duplicate request throttled: %s", fingerprint)
				http.Error(w, "Too many duplicate requests", http.StatusTooManyRequests)
				return
			}
		}

		// Store/Update fingerprint
		f.recent.Store(fingerprint, now)

		next(w, r, ps)
	}
}

func (f *FingerprintThrottleFilter) computeFingerprint(r *http.Request) (string, error) {
	hasher := sha256.New()

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr // fallback
	}
	method := r.Method
	path := r.URL.Path

	// Basic fingerprint parts
	hasher.Write([]byte(ip))
	hasher.Write([]byte(method))
	hasher.Write([]byte(path))

	// Headers of interest
	headers := []string{"Authorization", "User-Agent", "Content-Type"}
	for _, h := range headers {
		hasher.Write([]byte(r.Header.Get(h)))
	}

	// Include body if available (e.g., POST/PUT)
	if method == http.MethodPost || method == http.MethodPut {
		// Ensure body is always closed, even if panic occurs
		defer r.Body.Close()

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return "", err
		}

		// Reconstruct Body so it can be read again downstream
		r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

		hasher.Write(bodyBytes)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (f *FingerprintThrottleFilter) cleanupOldEntries() {
	now := time.Now()
	f.recent.Range(func(key, value any) bool {
		if ts, ok := value.(time.Time); ok {
			if now.Sub(ts) > 2*throttleWindow {
				f.recent.Delete(key)
			}
		}
		return true
	})
}
