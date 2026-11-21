/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package engineguard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

// Engine-native security adapter (S3, DESIGN §S3.1): the policy stays
// defined in coco (roles, sharing, field restrictions); the engine
// carries a PROJECTION of it as dedicated coco_tier_<sig> users with
// role bodies combining DLS / FLS / field_mask — the mechanisms verified
// against Easysearch (role PUT _security/role/<n> with indices[].dls/fls/
// field_mask, run-as via the security_run_as header; see DESIGN §S3.0).
//
// v1 scope: tier signature, policy compilation, role/user application,
// capability probe and drift read-back. The read path stays app-layer —
// switching searches to security_run_as is the "enforcement" flip gated
// on settings, deliberately NOT part of this drop.

const (
	rolePrefix = "coco_tier_"
	// RunAsHeader is the verified impersonation header: the privileged
	// connection identity speaks FOR the tier user; the tier password
	// never leaves coco.
	RunAsHeader = "security_run_as"

	probeTimeout = 5 * time.Second
)

// Capabilities reports what the engine's security plugin can do right now.
type Capabilities struct {
	SecurityAPI bool   `json:"security_api"` // _security/role readable
	RunAs       bool   `json:"run_as"`       // security_run_as accepted
	CheckedAt   int64  `json:"checked_at"`
	Reason      string `json:"reason,omitempty"`
}

// client is a thin REST caller against the engine's security API.
type client struct {
	baseURL  string
	username string
	password string
	http     *http.Client
}

// secClient builds the caller from the configured system engine.
// Returns nil when the engine config is unusable (probe then reports why).
var secClientFn = newSecClient

func newSecClient() *client {
	cfg := elastic.GetConfigNoPanic(global.MustLookupString(elastic.GlobalSystemElasticsearchID))
	if cfg == nil {
		return nil
	}
	endpoint := cfg.Endpoint
	if endpoint == "" && len(cfg.Endpoints) > 0 {
		endpoint = cfg.Endpoints[0]
	}
	if endpoint == "" {
		return nil
	}
	username, password := "", ""
	if cfg.BasicAuth != nil {
		username = cfg.BasicAuth.Username
		password = string(cfg.BasicAuth.Password)
	}
	return &client{
		baseURL:  strings.TrimSuffix(endpoint, "/"),
		username: username,
		password: password,
		http:     &http.Client{Timeout: probeTimeout},
	}
}

func (c *client) do(method, path string, body []byte) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return nil, 0, err
	}
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return out, resp.StatusCode, err
}

// Probe checks the security API surface (read one well-known role) and
// the run-as acceptance (a tiny search with the header, expecting a
// permission denial rather than an unknown-header success).
func Probe() Capabilities {
	c := secClientFn()
	if c == nil {
		return Capabilities{Reason: "engine config unavailable"}
	}
	caps := Capabilities{CheckedAt: time.Now().UnixMilli()}
	if _, code, err := c.do(http.MethodGet, "/_security/role/admin", nil); err == nil && (code == 200 || code == 404) {
		caps.SecurityAPI = true
	} else if err != nil {
		caps.Reason = fmt.Sprintf("security api probe: %v", err)
	} else {
		caps.Reason = fmt.Sprintf("security api returned %d", code)
	}
	if caps.SecurityAPI {
		// run-as acceptance: asking to run as a (very likely) nonexistent
		// user must produce an auth-style rejection; a 200 means the
		// header was ignored outright
		req, _ := http.NewRequest(http.MethodGet, c.baseURL+"/coco_document-v2/_search?size=0", nil)
		req.SetBasicAuth(c.username, c.password)
		req.Header.Set(RunAsHeader, "coco_tier_probe_nonexistent")
		if resp, err := c.http.Do(req); err == nil {
			resp.Body.Close()
			caps.RunAs = resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden
			if !caps.RunAs {
				caps.Reason = fmt.Sprintf("run-as probe returned %d (header ignored?)", resp.StatusCode)
			}
		}
	}
	return caps
}

// ApplyTierRole PUTs the tier role; ApplyTierUser PUTs the tier user.
func ApplyTierRole(roleName string, body map[string]interface{}) error {
	c := secClientFn()
	if c == nil {
		return fmt.Errorf("engine config unavailable")
	}
	raw, err := util.ToJSONBytes(body)
	if err != nil {
		return err
	}
	_, code, err := c.do(http.MethodPut, "/_security/role/"+roleName, raw)
	if err != nil {
		return err
	}
	if code != 200 && code != 201 {
		return fmt.Errorf("role apply returned %d", code)
	}
	return nil
}

func ApplyTierUser(userName, password string) error {
	c := secClientFn()
	if c == nil {
		return fmt.Errorf("engine config unavailable")
	}
	body := map[string]interface{}{"password": password, "roles": []string{userName}}
	raw, _ := util.ToJSONBytes(body)
	_, code, err := c.do(http.MethodPut, "/_security/user/"+userName, raw)
	if err != nil {
		return err
	}
	if code != 200 && code != 201 {
		return fmt.Errorf("user apply returned %d", code)
	}
	return nil
}

// FetchTierRole reads a role back (drift comparison).
func FetchTierRole(roleName string) (map[string]interface{}, error) {
	c := secClientFn()
	if c == nil {
		return nil, fmt.Errorf("engine config unavailable")
	}
	out, code, err := c.do(http.MethodGet, "/_security/role/"+roleName, nil)
	if err != nil {
		return nil, err
	}
	if code == 404 {
		return nil, nil
	}
	if code != 200 {
		return nil, fmt.Errorf("role fetch returned %d", code)
	}
	wrapper := map[string]json.RawMessage{}
	if err := json.Unmarshal(out, &wrapper); err != nil {
		return nil, err
	}
	raw, ok := wrapper[roleName]
	if !ok {
		return nil, nil
	}
	role := map[string]interface{}{}
	return role, json.Unmarshal(raw, &role)
}

/* ---------------- tier signature & policy compilation (pure) ---------------- */

// TierFacets is the effective permission shape one user resolves to.
type TierFacets struct {
	DatasourceIDs []string // visible source ids (DLS terms)
	OwnerID       string   // own documents always visible
	FieldExcludes []string // FLS excludes (payload, chunks, ...)
	MaskRules     []MaskRule
	UserAttrs     []string // attribute-based DLS terms (dept/clearance…)
}

// MaskRule compiles to one field_mask entry (regex pipeline or hash).
type MaskRule struct {
	Field   string `json:"field"`
	Action  string `json:"action"` // hash | regex
	Pattern string `json:"pattern,omitempty"`
	Replace string `json:"replace,omitempty"`
}

// TierSignature is the deterministic hash of the facets — users with the
// same signature share one engine tier user.
func TierSignature(f TierFacets) string {
	parts := []string{}
	add := func(ss ...string) { parts = append(parts, strings.Join(ss, "=")) }
	sorted := append([]string{}, f.DatasourceIDs...)
	sort.Strings(sorted)
	add("ds", strings.Join(sorted, ","))
	add("owner", f.OwnerID)
	fe := append([]string{}, f.FieldExcludes...)
	sort.Strings(fe)
	add("fls", strings.Join(fe, ","))
	add("mask", MaskRulesKey(f.MaskRules))
	attrs := append([]string{}, f.UserAttrs...)
	sort.Strings(attrs)
	add("attrs", strings.Join(attrs, ","))
	return util.MD5digest(strings.Join(parts, "|"))
}

// MaskRulesKey gives mask rules a stable ordering inside the signature.
func MaskRulesKey(rules []MaskRule) string {
	keys := make([]string, 0, len(rules))
	for _, r := range rules {
		keys = append(keys, fmt.Sprintf("%s:%s:%s:%s", r.Field, r.Action, r.Pattern, r.Replace))
	}
	sort.Strings(keys)
	return strings.Join(keys, ";")
}

// TierUserName derives the engine user name for a signature.
func TierUserName(sig string) string {
	if len(sig) > 8 {
		sig = sig[:8]
	}
	return rolePrefix + sig
}

// CompileRoleBody builds the role JSON (DESIGN §S3.4 shape): DLS over
// source.id terms + owner, FLS excludes, field_mask entries.
func CompileRoleBody(f TierFacets, indices []string) map[string]interface{} {
	should := []interface{}{}
	if len(f.DatasourceIDs) > 0 {
		terms := map[string]interface{}{"terms": map[string]interface{}{"source.id": f.DatasourceIDs}}
		should = append(should, terms)
	}
	if f.OwnerID != "" {
		should = append(should, map[string]interface{}{"term": map[string]interface{}{"_system.owner_id": f.OwnerID}})
	}
	if len(f.UserAttrs) > 0 {
		should = append(should, map[string]interface{}{"terms": map[string]interface{}{"_system.attrs": f.UserAttrs}})
	}
	dls := ""
	if len(should) > 0 {
		if raw, err := util.ToJSONBytes(map[string]interface{}{"bool": map[string]interface{}{"should": should}}); err == nil {
			dls = string(raw)
		}
	}

	fls := make([]string, 0, len(f.FieldExcludes))
	for _, x := range f.FieldExcludes {
		fls = append(fls, "~"+x) // exclude convention from the S3.0 probe
	}
	masks := make([]string, 0, len(f.MaskRules))
	for _, r := range f.MaskRules {
		switch r.Action {
		case "hash":
			masks = append(masks, r.Field)
		default:
			masks = append(masks, fmt.Sprintf("%s::%s::%s", r.Field, r.Pattern, r.Replace))
		}
	}

	indexBlock := map[string]interface{}{
		"names":      indices,
		"privileges": []string{"read"},
	}
	if dls != "" {
		indexBlock["dls"] = dls
	}
	if len(fls) > 0 {
		indexBlock["fls"] = fls
	}
	if len(masks) > 0 {
		indexBlock["field_mask"] = masks
	}
	return map[string]interface{}{
		"cluster": []string{},
		"indices": []interface{}{indexBlock},
	}
}
