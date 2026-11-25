/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeFolderPath(t *testing.T) {
	// canonicalization: leading slash added, trailing dropped
	p, err := NormalizeFolderPath("部门/研发/")
	require.NoError(t, err)
	assert.Equal(t, "/部门/研发", p)

	// root forms collapse to empty
	for _, in := range []string{"", "/", "  "} {
		p, err = NormalizeFolderPath(in)
		require.NoError(t, err)
		assert.Empty(t, p)
	}

	// rejections: traversal, empty segment, over-deep, over-long
	for _, bad := range []string{"/a/../b", "/a/./b", "/a//b"} {
		_, err = NormalizeFolderPath(bad)
		assert.Error(t, err, "%q must be rejected", bad)
	}
	long := ""
	for i := 0; i < folderMaxSegment+1; i++ {
		long += "x"
	}
	_, err = NormalizeFolderPath("/" + long)
	assert.Error(t, err)
	deep := "a"
	for i := 0; i < folderMaxDepth+1; i++ {
		deep += "/a"
	}
	_, err = NormalizeFolderPath(deep)
	assert.Error(t, err)
}
