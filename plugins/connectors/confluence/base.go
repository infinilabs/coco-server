/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package confluence

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
)

const (
	PathContentSearch = "/rest/api/content/search"
)

// ConfluenceHandler provides confluence APIs operation
type ConfluenceHandler struct {
	endpoint *url.URL
	Client   *http.Client
	username string
	token    string
}

// WithAuthOption supports unauthenticated access to confluence.
// if username and token are not set, do not add authorization header
func (handler *ConfluenceHandler) WithAuthOption(req *http.Request) {
	if handler.username != "" && handler.token != "" {
		req.SetBasicAuth(handler.username, handler.token)
	} else if handler.token != "" {
		req.Header.Set("Authorization", "Bearer "+handler.token)
	}
}

// NewConfluenceHandler construct ConfluenceHandler
func NewConfluenceHandler(location string, username string, token string) (*ConfluenceHandler, error) {
	endpoint, err := url.ParseRequestURI(location)

	if err != nil {
		return nil, err
	}

	handler := &ConfluenceHandler{}
	handler.endpoint = endpoint
	handler.username = username
	handler.token = token

	// #nosec G402
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}

	handler.Client = &http.Client{Transport: tr}
	return handler, nil
}

// Request implements the basic Request function
func (handler *ConfluenceHandler) Request(req *http.Request) ([]byte, error) {
	req.Header.Add("Accept", "application/json, */*")

	handler.WithAuthOption(req)

	log.Debugf("[confluence] request uri: [%v]", req.RequestURI)

	if global.Env().IsDebug {
		requestDump, err := httputil.DumpRequest(req, true)
		if err != nil {
			log.Debugf("[confluence] dump http request failed: %v", err)
		}
		log.Debugf("[confluence] request: [%v]", string(requestDump))
	}

	resp, err := handler.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	log.Debugf("[confluence] response code: [%v]", resp.StatusCode)

	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if global.Env().IsDebug {
		log.Debugf("[confluence] response: [%v]", string(res))
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusPartialContent:
		return res, nil
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("authentication failed, status: %s", resp.Status)
	case http.StatusForbidden:
		return nil, fmt.Errorf("permission forbidden, status: %s", resp.Status)
	}

	return nil, fmt.Errorf("confluence server response error, status: %s", resp.Status)
}

func (handler *ConfluenceHandler) SearchContent(ctx context.Context, query SearchContentRequest, options ...QueryOption) (*SearchContentResponse, error) {
	ep, err := url.ParseRequestURI(handler.endpoint.String() + PathContentSearch)
	if err != nil {
		return nil, err
	}
	ep.RawQuery = QueryParamsOptions(BuildSearchContentQueryParamsFrom(query), options...).Encode()
	return handler.fetchContentResults(ctx, ep, http.MethodGet)
}

func (handler *ConfluenceHandler) SearchNextContent(ctx context.Context, next string) (*SearchContentResponse, error) {
	ep, err := url.ParseRequestURI(handler.endpoint.String() + next)
	if err != nil {
		return nil, err
	}
	return handler.fetchContentResults(ctx, ep, http.MethodGet)
}

func (handler *ConfluenceHandler) fetchContentResults(ctx context.Context, ep *url.URL, method string) (*SearchContentResponse, error) {
	req, err := http.NewRequestWithContext(ctx, method, ep.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := handler.Request(req)
	if err != nil {
		return nil, err
	}

	var search SearchContentResponse

	err = json.Unmarshal(res, &search)
	if err != nil {
		return nil, err
	}

	return &search, nil
}

// BuildSearchContentQueryParamsFrom build query parameters from SearchContentRequest
func BuildSearchContentQueryParamsFrom(query SearchContentRequest) *url.Values {
	p := &url.Values{}
	WithExpand(query.Expand...)(p)
	WithLimit(query.Limit)(p)
	WithStart(query.Start)(p)
	WithCQL(query.CQL)(p)
	WithCQLContext(query.CQLContext)(p)
	return p
}

type CqlBuilder struct {
	strings.Builder
}

func NewCqlBuilder() *CqlBuilder {
	return &CqlBuilder{strings.Builder{}}
}

func (b *CqlBuilder) WithType(typeNames ...string) *CqlBuilder {
	if len(typeNames) > 0 {
		b.WriteString(" type in (")
		b.WriteString(strings.Join(append([]string{}, typeNames...), ","))
		b.WriteRune(')')
	}
	return b
}

func (b *CqlBuilder) WithSpace(spaces ...string) *CqlBuilder {
	if len(spaces) > 0 {
		b.WriteString(" space in (")
		b.WriteString(strings.Join(append([]string{}, spaces...), ","))
		b.WriteRune(')')
	}
	return b
}

func (b *CqlBuilder) And() *CqlBuilder {
	b.WriteString(" AND ")
	return b
}

func (b *CqlBuilder) String() string {
	return b.Builder.String()
}
