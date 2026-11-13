/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"regexp"
	"strings"
	"sync"

	"infini.sh/coco/core"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

// Dynamic data masking and field-level access control.
//
// The three permission tiers stack: which datasources (indexes) a user may
// search is decided by sharing; which documents inside them survive is
// decided by document sharing plus owner rules; which fields remain in what
// is returned is decided here, by role. Masking then guards the last exit:
// recalled content that is about to leave for a model provider gets regex
// scrubbing regardless of tier — the LLM never sees the raw values.

var maskingRegexCache sync.Map // pattern string -> *regexp.Regexp (nil on compile error)

func compileMaskPattern(pattern string) *regexp.Regexp {
	if cached, ok := maskingRegexCache.Load(pattern); ok {
		re, _ := cached.(*regexp.Regexp)
		return re
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		re = nil
	}
	maskingRegexCache.Store(pattern, re)
	return re
}

// MaskContent applies every enabled masking rule, in configured order, to
// text that is about to be handed to a model. With masking disabled or no
// rules matching, the text passes through unchanged.
func MaskContent(text string) string {
	cfg := AppConfig().DataSecurity
	if cfg == nil || cfg.Masking == nil {
		return text
	}
	return applyMaskingRules(text, cfg.Masking)
}

// applyMaskingRules is the testable core of MaskContent.
func applyMaskingRules(text string, settings *core.MaskingSettings) string {
	if settings == nil || !settings.Enabled || text == "" {
		return text
	}
	for _, rule := range settings.Rules {
		if !rule.Enabled || rule.Pattern == "" {
			continue
		}
		re := compileMaskPattern(rule.Pattern)
		if re == nil {
			continue
		}
		text = re.ReplaceAllString(text, rule.Replacement)
	}
	return text
}

// ActiveMaskingRules returns the enabled rules; used by the settings UI to
// preview the exact set the server would apply.
func ActiveMaskingRules() []core.MaskingRule {
	cfg := AppConfig().DataSecurity
	if cfg == nil || cfg.Masking == nil {
		return nil
	}
	out := []core.MaskingRule{}
	for _, rule := range cfg.Masking.Rules {
		if rule.Enabled {
			out = append(out, rule)
		}
	}
	return out
}

// RestrictedFieldsForRoles returns the union of excluded field patterns for
// the given roles. Admins are never restricted.
func RestrictedFieldsForRoles(roles []string) []string {
	cfg := AppConfig().DataSecurity
	if cfg == nil {
		return nil
	}
	return restrictedFieldsFor(cfg.FieldAccess, roles)
}

func restrictedFieldsFor(fieldAccess *core.FieldAccessSettings, roles []string) []string {
	if fieldAccess == nil || len(fieldAccess.Restrictions) == 0 {
		return nil
	}
	for _, role := range roles {
		if role == security.RoleAdmin {
			return nil
		}
	}
	seen := map[string]struct{}{}
	out := []string{}
	for _, restriction := range fieldAccess.Restrictions {
		if restriction.Role == "" {
			continue
		}
		matched := false
		for _, role := range roles {
			if role == restriction.Role {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		for _, field := range restriction.ExcludeFields {
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}
			if _, dup := seen[field]; !dup {
				seen[field] = struct{}{}
				out = append(out, field)
			}
		}
	}
	return out
}

// RemoveRestrictedFields strips restricted fields from a decoded document
// source map, mirroring ES _source exclude semantics: "payload" removes the
// whole field, "payload.*" empties an object field while leaving scalars
// (whose sub-parts do not exist) untouched. Returns true when the map changed.
func RemoveRestrictedFields(source util.MapStr, roles []string) bool {
	return removeRestrictedFields(source, RestrictedFieldsForRoles(roles))
}

func removeRestrictedFields(source util.MapStr, patterns []string) bool {
	if len(patterns) == 0 || len(source) == 0 {
		return false
	}
	changed := false
	for _, pattern := range patterns {
		root := strings.TrimSuffix(pattern, ".*")
		value, ok := source[root]
		if !ok {
			continue
		}
		if root == pattern {
			source.Delete(root)
			changed = true
			continue
		}
		if sub, ok := value.(map[string]interface{}); ok && len(sub) > 0 {
			source[root] = map[string]interface{}{}
			changed = true
		}
	}
	return changed
}
