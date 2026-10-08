/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

func TestCompileFAQContentExcludesNegativesAndAnswer(t *testing.T) {
	entry := &FAQEntry{
		Standard: "如何重置密码",
		Similar:  []string{"忘记密码怎么办", "password reset"},
		Negative: []string{"如何重置数据库"},
		Answer:   "在设置-安全里点击重置",
	}
	content := compileFAQContent(entry)
	assert.Contains(t, content, "如何重置密码")
	assert.Contains(t, content, "忘记密码怎么办")
	assert.Contains(t, content, "password reset")
	assert.NotContains(t, content, "如何重置数据库", "negative questions must NEVER index")
	assert.NotContains(t, content, "在设置-安全里点击重置", "answers are returned, not searched")
}

func TestFAQQuestionSetNormalization(t *testing.T) {
	entry := &FAQEntry{Standard: "How do I  RESET ?", Similar: []string{"reset password"}}
	set := faqQuestionSet(entry)
	// full-width/half-width, case, whitespace collapse — the D1 chain
	_, ok := set[fingerprintNormalizeForTest("ｈｏｗ  do i reset?")]
	assert.True(t, ok, "normalized phrasings compare equal across widths and case")
	require.Len(t, set, 2)
}

func TestFAQNegativeHit(t *testing.T) {
	entry := &FAQEntry{
		Standard: "年假有几天",
		Negative: []string{"事假有几天", "病假呢"},
	}
	assert.True(t, faqNegativeHit(entry, fingerprintNormalizeForTest("事假 有 几天")), "negative exact hit knocks the entry out")
	assert.False(t, faqNegativeHit(entry, fingerprintNormalizeForTest("年假几天")))
}

func TestFAQDocumentCarriesStructuredEntry(t *testing.T) {
	entry := &FAQEntry{
		Standard: "报销流程",
		Similar:  []string{"怎么报销"},
		Answer:   "OA 提交",
	}
	doc := faqDocument(entry, "ds-faq", nil)
	assert.Equal(t, faqDocType, doc.Type)
	assert.Equal(t, "报销流程", doc.Title)
	assert.Contains(t, doc.Content, "怎么报销")
	assert.NotEmpty(t, doc.ContentHash, "pre-stamped so the duplicate check agrees with the persisted row")

	stored := faqEntryOf(doc)
	require.NotNil(t, stored)
	assert.Equal(t, "OA 提交", stored.Answer)
	assert.Equal(t, []string{"怎么报销"}, stored.Similar)
}

// fingerprintNormalizeForTest mirrors normalizeFAQQuestion through the
// exported fingerprint chain (same function, named for readability here).
func fingerprintNormalizeForTest(s string) string {
	return normalizeFAQQuestion(s)
}

var _ = core.DocumentStatusCompleted // keep core import stable for helpers above
