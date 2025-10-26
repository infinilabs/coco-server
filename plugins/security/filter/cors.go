/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package filter

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"net/http"
	"sync"
)

func init() {
	api.RegisterUIFilter(&CORSFilter{})
}

type CORSFilter struct {
	api.Handler
}

func (f *CORSFilter) GetPriority() int {
	return 100
}

const FeatureCORS = "feature_cors"
const FeatureNotAllowCredentials = "feature_not_allow_credentials"

func (f *CORSFilter) ApplyFilter(
	method string,
	pattern string,
	options *api.HandlerOptions,
	next httprouter.Handle,
) httprouter.Handle {

	//option not enabled
	if options == nil || !options.Feature(FeatureCORS) {
		log.Debug(method, ",", pattern, ",skip feature ", FeatureCORS)
		return next
	}

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		origin := r.Header.Get("Origin")
		if options.Feature(core.FeatureByPassCORSCheck) || (origin != "" && (r.Method == http.MethodOptions || isAllowedOrigin(origin, r))) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-TOKEN, APP-INTEGRATION-ID, WEBSOCKET-SESSION-ID")
			if options.Feature(FeatureNotAllowCredentials) {
				w.Header().Set("Access-Control-Allow-Credentials", "false")
			} else {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			// Handle preflight (OPTIONS) requests
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				// Respond with 200 OK for OPTIONS requests
				w.WriteHeader(http.StatusOK)
				return
			}
		} else {
			if global.Env().IsDebug {
				log.Warn("skipping place CORS headers: ", method, ",", pattern, ",origin:", origin, ",", origin != "", ",", r.Method == http.MethodOptions, ",", isAllowedOrigin(origin, r))
			}
		}

		next(w, r, ps)
	}
}

var (
	allowOriginFuncs sync.Map
)

// RegisterAllowOriginFunc registers a function to check if the origin is allowed.
// The key is used to identify the function.
func RegisterAllowOriginFunc(key string, fn AllowOriginFunc) {
	if _, exists := allowOriginFuncs.Load(key); exists {
		panic("key already exists, maybe you can remove it first")
	}
	allowOriginFuncs.Store(key, fn)
}

// RemoveAllowOriginFunc removes the function to check if the origin is allowed.
func RemoveAllowOriginFunc(key string) {
	allowOriginFuncs.Delete(key)
}

// AllowOriginFunc is a function that checks if the origin is allowed.
type AllowOriginFunc func(origin string, req *http.Request) bool

func isAllowedOrigin(origin string, req *http.Request) bool {
	isAllowed := false
	allowOriginFuncs.Range(func(key, value interface{}) bool {
		if fn, ok := value.(AllowOriginFunc); ok {
			//note: hear we pass the request to the function to allow the function to implement more complex logic
			if fn != nil && fn(origin, req) {
				isAllowed = true
				// break the loop
				return false
			}
		}
		return true
	})
	return isAllowed
}
