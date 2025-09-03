/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */
package filter

import (
	"bytes"
	"encoding/json"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"net/http"
)

const FeatureMaskSensitiveField = "feature_sensitive_fields"
const FeatureRemoveSensitiveField = "feature_sensitive_fields_remove_sensitive_field"

const SensitiveFields = "feature_sensitive_fields_extra_keys"

var sensitiveFields = map[string]bool{
	"password":      true,
	"token":         true,
	"secret":        true,
	"access_token":  true,
	"refresh_token": true,
}

type JSONMaskFilter struct{}

func init() {
	api.RegisterUIFilter(&JSONMaskFilter{})
}

func (f *JSONMaskFilter) GetPriority() int {
	// Lower values execute first
	return 1000
}

func (f *JSONMaskFilter) ApplyFilter(
	method string,
	pattern string,
	options *api.HandlerOptions,
	next httprouter.Handle,
) httprouter.Handle {

	//option not enabled
	if options == nil || !(options.Feature(FeatureRemoveSensitiveField) || options.Feature(FeatureMaskSensitiveField)) {
		log.Debug(method, ",", pattern, ",skip feature ", FeatureMaskSensitiveField, ",", options.Feature(FeatureRemoveSensitiveField), ",", options.Feature(FeatureMaskSensitiveField))
		return next
	}

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		// Create a response recorder to capture the response
		rec := &responseInterceptor{ResponseWriter: w, buf: new(bytes.Buffer)}
		next(rec, r, ps)

		var extraFields map[string]bool
		extra, ok := options.Labels[SensitiveFields]
		if ok {
			extraFields = map[string]bool{}
			extraFields, ok = extra.(map[string]bool)
		}

		var remove = options.Feature(FeatureRemoveSensitiveField)

		// Process and modify the response body
		maskedBody := maskJSONFields(rec.buf.Bytes(), extraFields, remove)

		// Write the modified response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rec.statusCode)
		w.Write(maskedBody)
	}
}

// Interceptor to capture the response body
type responseInterceptor struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	statusCode int
}

func (r *responseInterceptor) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseInterceptor) Write(b []byte) (int, error) {
	return r.buf.Write(b)
}

// Function to mask sensitive fields in JSON
func maskJSONFields(data []byte, secretFields map[string]bool, remove bool) []byte {
	var result map[string]interface{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return data // Return original data if parsing fails
	}

	maskFields(result, secretFields, remove)

	modified, err := json.Marshal(result)
	if err != nil {
		return data // Return original data if re-encoding fails
	}
	return modified
}
func maskFields(obj map[string]interface{}, secretFields map[string]bool, remove bool) {
	for key, value := range obj {
		// If the key is in the sensitive list, either remove it or mask it
		if _, found := sensitiveFields[key]; found || secretFields[key] {
			if remove {
				delete(obj, key)
			} else {
				obj[key] = "***"
			}
			continue
		}

		// Recursively process nested objects
		switch v := value.(type) {
		case map[string]interface{}:
			maskFields(v, secretFields, remove) // Recursive call for nested maps

		case []interface{}:
			newArray := make([]interface{}, 0, len(v)) // Create a new slice

			for _, item := range v {
				if nestedObj, ok := item.(map[string]interface{}); ok {
					maskFields(nestedObj, secretFields, remove) // Process objects inside arrays
					newArray = append(newArray, nestedObj)
				} else {
					// If the key is sensitive and the array contains raw values, mask or remove
					if secretFields[key] || sensitiveFields[key] {
						if !remove {
							newArray = append(newArray, "***") // Mask
						}
					} else {
						newArray = append(newArray, item) // Keep non-sensitive values
					}
				}
			}

			obj[key] = newArray // Replace with modified array
		}
	}
}
