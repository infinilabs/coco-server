/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package confluence

import (
	"net/url"
	"strconv"
	"strings"
)

// QueryOption configures the url.Values instance to add or modify query parameters.
type QueryOption func(params *url.Values)

// WithExpand add `expand` parameter, supports multiple values separated by commas
func WithExpand(expands ...string) QueryOption {
	return func(params *url.Values) {
		if len(expands) > 0 {
			params.Set("expand", strings.Join(expands, ","))
		}
	}
}

// WithLimit add `limit` parameter
func WithLimit(limit int) QueryOption {
	return func(params *url.Values) {
		if limit != 0 { // only if `limit` is not equal to zero
			params.Set("limit", strconv.Itoa(limit))
		}
	}
}

// WithStart add `start` parameter
func WithStart(start int) QueryOption {
	return func(params *url.Values) {
		if start != 0 {
			params.Set("start", strconv.Itoa(start))
		}
	}
}

// WithCQL add `CQL` (Confluence Query Language) parameter
// Confluence search API provides
func WithCQL(cql string) QueryOption {
	return func(params *url.Values) {
		if cql != "" {
			params.Set("cql", cql)
		}
	}
}

// WithCQLContext add `cqlcontext` parameter
// Confluence search API provides
func WithCQLContext(cqlContext string) QueryOption {
	return func(params *url.Values) {
		if cqlContext != "" {
			params.Set("cqlcontext", cqlContext)
		}
	}
}

func QueryParamsOptions(params *url.Values, options ...QueryOption) *url.Values {
	// for each Option func
	for _, opt := range options {
		opt(params)
	}
	return params
}
