/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package text_attachment_extraction

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildZip(t *testing.T, name string, entries map[string]string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	f, err := os.Create(p)
	require.NoError(t, err)
	defer f.Close()
	zw := zip.NewWriter(f)
	for entry, body := range entries {
		w, err := zw.Create(entry)
		require.NoError(t, err)
		_, err = w.Write([]byte(body))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return p
}

// ---- EPUB ----

func TestParseEpubSpineOrderAndTitle(t *testing.T) {
	p := buildZip(t, "book.epub", map[string]string{
		"mimetype": "application/epub+zip",
		"META-INF/container.xml": `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`,
		"OEBPS/content.opf": `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>设计手册</dc:title></metadata>
  <manifest>
    <item id="c2" href="ch2.xhtml" media-type="application/xhtml+xml"/>
    <item id="c1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="css" href="style.css" media-type="text/css"/>
  </manifest>
  <spine><itemref idref="c1"/><itemref idref="c2"/></spine>
</package>`,
		"OEBPS/ch1.xhtml": `<html><body><h1>第一章</h1><p>开工。</p></body></html>`,
		"OEBPS/ch2.xhtml": `<html><body><h1>第二章</h1><p>收尾。</p></body></html>`,
		"OEBPS/style.css": "body{color:red}",
	})

	pages, err := parseEpub(p)
	require.NoError(t, err)
	require.Len(t, pages, 2, "spine order decides pages; css never becomes one")
	assert.Contains(t, pages[0], "# 设计手册", "DC title leads the first page")
	assert.Contains(t, pages[0], "# 第一章")
	assert.Contains(t, pages[1], "# 第二章")
	assert.NotContains(t, pages[1], "设计手册", "title rides the first page only")
}

func TestParseEpubRejectsNonZip(t *testing.T) {
	_, err := parseEpub(writeMagicFile(t, "fake.epub", oleMagic))
	require.Error(t, err)
}

// ---- XMind ----

func TestParseXmindOutline(t *testing.T) {
	p := buildZip(t, "map.xmind", map[string]string{
		"content.json": `[
  {"title":"架构","rootTopic":{"title":"网关","notes":{"plain":{"content":"流量入口\n灰度开关"}},"children":{"attached":[
    {"title":"鉴权","children":{"attached":[{"title":"JWT"}]}},
    {"title":"限流"}
  ]}}},
  {"title":"","rootTopic":{"title":"孤立画布","children":{"attached":[]}}}
]`,
		"metadata.json": `{"creator":{"name":"test"}}`,
	})

	pages, err := parseXmind(p)
	require.NoError(t, err)
	require.Len(t, pages, 2)

	assert.Equal(t, "# 架构\n- 网关\n  > 流量入口\n  > 灰度开关\n  - 鉴权\n    - JWT\n  - 限流", pages[0])
	assert.Equal(t, "# Sheet 2\n- 孤立画布", pages[1], "untitled sheet falls back to Sheet N")
}

func TestParseXmindRejectsLegacyLayout(t *testing.T) {
	p := buildZip(t, "old.xmind", map[string]string{"content.xml": "<xmap></xmap>"})
	_, err := parseXmind(p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "legacy content.xml")
}

// ---- MHTML ----

const fxMhtml = `From: <saved@example.com>
Subject: Snapshot
Content-Type: multipart/related; boundary="BOUND"

--BOUND
Content-Type: text/html; charset=utf-8
Content-Location: https://ads.googleads.example/x
Content-Transfer-Encoding: base64

` + "PGh0bWw+PGJvZHk+PHAxPmFkIGNvcHk8L3A+PC9ib2R5PjwvaHRtbD4=" + `

--BOUND
Content-Type: text/html; charset=utf-8
Content-Location: https://example.com/page
Content-Transfer-Encoding: quoted-printable

<html><body><h1>=E6=8A=A5=E5=91=8A</h1><p>quarterly numbers</p></body></html>
--BOUND--
`

func TestParseMhtmlPicksLargestNonAdHtml(t *testing.T) {
	pages, err := parseMhtml([]byte(fxMhtml))
	require.NoError(t, err)
	require.Len(t, pages, 1)
	assert.Contains(t, pages[0], "quarterly numbers")
	assert.Contains(t, pages[0], "报告", "quoted-printable body decodes")
	assert.NotContains(t, pages[0], "ad copy", "googleads part is skipped")
}

func TestParseMhtmlRejectsNonMultipart(t *testing.T) {
	_, err := parseMhtml([]byte("Content-Type: text/plain\r\n\r\nhello"))
	require.Error(t, err)
}

// ---- shared html->text ----

func TestHtmlToTextStructureMarkers(t *testing.T) {
	out, err := htmlToText(strings.NewReader(`<html><head><style>.x{}</style></head>
<body><script>bad()</script><h2>季度报告</h2><table><tr><th>指标</th><th>值</th></tr><tr><td>收入</td><td>100</td></tr></table><p>结论<em>如上</em>。</p></body></html>`))
	require.NoError(t, err)
	assert.Contains(t, out, "## 季度报告", "h2 becomes a markdown level-2 heading")
	assert.Contains(t, out, "指标 | 值")
	assert.Contains(t, out, "收入 | 100", "table rows stay one line, cells joined")
	assert.Contains(t, out, "结论如上。", "inline text joins inside paragraphs (no injected spaces)")
	assert.NotContains(t, out, "bad()", "script dropped")
	assert.NotContains(t, out, ".x{}", "style dropped")
}
