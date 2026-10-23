/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://www.infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func zipForTest(t *testing.T, entries map[string]string) *zip.Reader {
	t.Helper()
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	return zr
}

func TestParseSkillPack(t *testing.T) {
	zr := zipForTest(t, map[string]string{
		"my-skill/SKILL.md":            "---\nname: my-skill\ndescription: pack import probe\n---\n\n# My Skill\n",
		"my-skill/references/api.md":   "api notes",
		"my-skill/references/deep.csv": "1,2",
		"my-skill/scripts/run.sh":      "echo hi",
		"my-skill/README.md":           "not a load", // outside references/ and scripts/
		"my-skill/.DS_Store":           "junk",
		"__MACOSX/my-skill/SKILL.md":   "mac noise",
	})
	doc, prefix, refs, err := parseSkillPack(zr)
	require.NoError(t, err)
	assert.Equal(t, "my-skill/", prefix)
	name, desc, body, ok := parseSkillFrontmatter(doc)
	require.True(t, ok)
	assert.Equal(t, "my-skill", name)
	assert.Equal(t, "pack import probe", desc)
	assert.Contains(t, body, "# My Skill")

	paths := make([]string, 0, len(refs))
	for _, r := range refs {
		paths = append(paths, r.relPath)
	}
	assert.ElementsMatch(t, []string{"references/api.md", "references/deep.csv", "scripts/run.sh"}, paths,
		"references/ and scripts/ ride along; README, dotfiles and __MACOSX stay out")
}

func TestParseSkillPackFlatAndErrors(t *testing.T) {
	// flat layout: SKILL.md at the zip root
	zr := zipForTest(t, map[string]string{
		"SKILL.md":               "---\nname: flat\n---\nbody",
		"references/notes.md":    "notes",
	})
	doc, prefix, refs, err := parseSkillPack(zr)
	require.NoError(t, err)
	assert.Equal(t, "", prefix)
	assert.Contains(t, doc, "name: flat")
	assert.Len(t, refs, 1)

	// no SKILL.md anywhere usable
	zr = zipForTest(t, map[string]string{"references/a.md": "a"})
	_, _, _, err = parseSkillPack(zr)
	assert.ErrorContains(t, err, "no SKILL.md")

	// SKILL.md nested two folders deep is not a package root
	zr = zipForTest(t, map[string]string{"a/b/SKILL.md": "---\nname: x\n---\n"})
	_, _, _, err = parseSkillPack(zr)
	assert.ErrorContains(t, err, "no SKILL.md")

	// two SKILL.md candidates at depth <= 1 is ambiguous
	zr = zipForTest(t, map[string]string{
		"SKILL.md":        "---\nname: x\n---\n",
		"other/SKILL.md":  "---\nname: y\n---\n",
	})
	_, _, _, err = parseSkillPack(zr)
	assert.ErrorContains(t, err, "multiple SKILL.md")
}
