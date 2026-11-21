/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"infini.sh/coco/modules/common"
)

// documentSourceExcludes is the single definition of which fields leave the
// engine on search paths: the fixed performance excludes (fields the frontend
// never renders and that bloat responses), plus the role-based field-access
// tier from data-security settings. Every read path that returns documents to
// a user must go through this (or, when _source excludes cannot be applied,
// through common.RemoveRestrictedFields — see getDoc).
func documentSourceExcludes(roles []string) []string {
	excludes := []string{"payload.*", "document_chunk", "ai_insights.embedding"}
	excludes = append(excludes, common.RestrictedFieldsForRoles(roles)...)
	return excludes
}
