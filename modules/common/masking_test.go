/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"testing"

	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

func TestApplyMaskingRules(t *testing.T) {
	// rules apply in configured order: the ID-card rule must come before the
	// phone rule, otherwise the phone pattern would chew into ID numbers
	settings := &core.MaskingSettings{
		Enabled: true,
		Rules: []core.MaskingRule{
			{ID: "2", Name: "id-card", Pattern: `\d{17}[\dXx]`, Replacement: "***ID***", Enabled: true},
			{ID: "1", Name: "phone", Pattern: `1[3-9]\d{9}`, Replacement: "***PHONE***", Enabled: true},
			{ID: "3", Name: "disabled", Pattern: `\d+`, Replacement: "NOPE", Enabled: false},
		},
	}

	text := "contact 13812345678, card 11010119900307851X, ref 42"
	got := applyMaskingRules(text, settings)
	want := "contact ***PHONE***, card ***ID***, ref 42"
	if got != want {
		t.Fatalf("masked = %q, want %q", got, want)
	}

	// global switch off -> passthrough
	off := &core.MaskingSettings{Enabled: false, Rules: settings.Rules}
	if got := applyMaskingRules(text, off); got != text {
		t.Fatalf("disabled masking should pass through, got %q", got)
	}

	// nil settings -> passthrough
	if got := applyMaskingRules(text, nil); got != text {
		t.Fatalf("nil settings should pass through, got %q", got)
	}

	// invalid regex is skipped, not fatal
	broken := &core.MaskingSettings{Enabled: true, Rules: []core.MaskingRule{
		{Pattern: "([invalid", Replacement: "x", Enabled: true},
		{Pattern: `1[3-9]\d{9}`, Replacement: "***PHONE***", Enabled: true},
	}}
	got = applyMaskingRules("call 13900000000", broken)
	if got != "call ***PHONE***" {
		t.Fatalf("broken rule should be skipped, got %q", got)
	}
}

func TestRestrictedFieldsForRoles(t *testing.T) {
	cfg := &core.FieldAccessSettings{Restrictions: []core.FieldRestriction{
		{Role: "guest", ExcludeFields: []string{"payload.*", "raw_content"}},
		{Role: "intern", ExcludeFields: []string{"payload.*", "summary"}},
	}}

	got := restrictedFieldsFor(cfg, []string{"guest"})
	if len(got) != 2 {
		t.Fatalf("guest should get 2 patterns, got %v", got)
	}

	// multiple roles union, dedup
	got = restrictedFieldsFor(cfg, []string{"guest", "intern"})
	if len(got) != 3 {
		t.Fatalf("guest+intern union should be 3 unique patterns, got %v", got)
	}

	// unmatched role -> nothing
	if got := restrictedFieldsFor(cfg, []string{"employee"}); len(got) != 0 {
		t.Fatalf("unmatched role should get nothing, got %v", got)
	}

	// admin is never restricted
	if got := restrictedFieldsFor(cfg, []string{"guest", "admin"}); got != nil {
		t.Fatalf("admin must bypass field restrictions, got %v", got)
	}

	// nil settings
	if got := restrictedFieldsFor(nil, []string{"guest"}); got != nil {
		t.Fatalf("nil settings should return nil, got %v", got)
	}
}

func TestRemoveRestrictedFields(t *testing.T) {
	source := util.MapStr{
		"title":   "quarterly report",
		"summary": "ok",
		"payload": map[string]interface{}{"secret": "s3cr3t", "size": 1},
		"chunks":  []interface{}{map[string]interface{}{"text": "body"}},
	}

	// "payload.*" empties the object, "summary" disappears entirely
	changed := removeRestrictedFields(source, []string{"payload.*", "summary"})
	if !changed {
		t.Fatal("source should have changed")
	}
	if _, exists := source["summary"]; exists {
		t.Fatal("summary should be removed")
	}
	payload, _ := source["payload"].(map[string]interface{})
	if len(payload) != 0 {
		t.Fatalf("payload should be emptied, got %v", payload)
	}
	if source["title"] != "quarterly report" || source["chunks"] == nil {
		t.Fatal("unrestricted fields must survive")
	}

	// no patterns -> unchanged
	untouched := util.MapStr{"title": "x"}
	if removeRestrictedFields(untouched, nil) {
		t.Fatal("no patterns should leave the map untouched")
	}

	// "payload.*" on a scalar leaves it alone (ES semantics: no sub-fields)
	scalar := util.MapStr{"payload": "plain"}
	if removeRestrictedFields(scalar, []string{"payload.*"}) {
		t.Fatal("scalar under x.* should be kept")
	}
}
