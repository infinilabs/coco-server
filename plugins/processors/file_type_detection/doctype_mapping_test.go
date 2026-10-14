/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package file_type_detection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// W2 L0.5: the new formats must carry a content category, not fall through
// to "" — uncategorized documents are invisible to category filters.
func TestContentTypeMappingCoversL05Formats(t *testing.T) {
	for ext, want := range map[string]string{
		".epub":           "doc",
		".xmind":          "doc",
		".mhtml":          "doc",
		".mht":            "doc",
		".html":           "doc",
		".htm":            "doc",
		".pdf":            "doc",
		".xlsx":           "doc",
		".jpg":            "image",
		".mp4":            "video",
		".unknownexttest": "",
	} {
		ct := getContentType(ext)
		assert.Equal(t, want, categorizeContentType(ct), "ext %s → content_type %q must categorize to %q", ext, ct, want)
	}
}
