/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package text_attachment_extraction

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- magic-number correction ----

func writeMagicFile(t *testing.T, name string, head []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, head, 0o644))
	return p
}

func TestEffectiveExtCorrectsMismatches(t *testing.T) {
	ole := append([]byte{}, oleMagic...)
	zipHead := append([]byte{}, zipMagic...)

	ext, note := effectiveExt(writeMagicFile(t, "final_v2.docx", ole), ".docx")
	assert.Equal(t, ".doc", ext)
	assert.Contains(t, note, "OLE Word")

	ext, note = effectiveExt(writeMagicFile(t, "slides.pptx", ole), ".pptx")
	assert.Equal(t, ".ppt-ole", ext, "OLE payload in OOXML name must avoid the native zip parser")
	assert.NotEmpty(t, note)

	ext, _ = effectiveExt(writeMagicFile(t, "report.xls", zipHead), ".xls")
	assert.Equal(t, ".xlsx", ext)

	ext, _ = effectiveExt(writeMagicFile(t, "deck.ppt", zipHead), ".ppt")
	assert.Equal(t, ".pptx", ext)

	// consistent files keep their extension, no note
	ext, note = effectiveExt(writeMagicFile(t, "plain.docx", zipHead), ".docx")
	assert.Equal(t, ".docx", ext)
	assert.Empty(t, note)

	// Tika-routed types never need correction
	ext, note = effectiveExt(writeMagicFile(t, "memo.doc", ole), ".doc")
	assert.Equal(t, ".doc", ext)
	assert.Empty(t, note)
}

// ---- structured xlsx route ----

const (
	fxWorkbook = `<?xml version="1.0"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"
           xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
    <sheet name="人员" sheetId="1" r:id="rId1"/>
    <sheet name="配置" sheetId="2" r:id="rId2"/>
  </sheets>
</workbook>`

	fxRels = `<?xml version="1.0"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/>
</Relationships>`

	// shared: 0=姓名 1=部门 2=张三 3=研发 4=李四 5=键 6=值 7=debug 8=true
	fxShared = `<?xml version="1.0"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <si><t>姓名</t></si><si><t>部门</t></si><si><t>张三</t></si>
  <si><t>研发</t></si><si><t>李四</t></si><si><t>键</t></si>
  <si><t>值</t></si><si><t>debug</t></si><si><t>true</t></si>
</sst>`

	// sheet1: header row; row2 张三/研发; row3 李四 with B3 covered by the
	// B2:B3 merge (only B2 stores the value)
	fxSheet1 = `<?xml version="1.0"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row>
    <row r="2"><c r="A2" t="s"><v>2</v></c><c r="B2" t="s"><v>3</v></c></row>
    <row r="3"><c r="A3" t="s"><v>4</v></c></row>
  </sheetData>
  <mergeCells count="1"><mergeCell ref="B2:B3"/></mergeCells>
</worksheet>`

	fxSheet2 = `<?xml version="1.0"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1"><c r="A1" t="s"><v>5</v></c><c r="B1" t="s"><v>6</v></c></row>
    <row r="2"><c r="A2" t="s"><v>7</v></c><c r="B2" t="s"><v>8</v></c></row>
  </sheetData>
</worksheet>`
)

func buildXlsx(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fixture.xlsx")
	f, err := os.Create(p)
	require.NoError(t, err)
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, body := range map[string]string{
		"xl/workbook.xml":            fxWorkbook,
		"xl/_rels/workbook.xml.rels": fxRels,
		"xl/sharedStrings.xml":       fxShared,
		"xl/worksheets/sheet1.xml":   fxSheet1,
		"xl/worksheets/sheet2.xml":   fxSheet2,
	} {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(body))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return p
}

func TestParseXlsxHeaderAndMergeFill(t *testing.T) {
	pages, err := parseXlsx(buildXlsx(t), true)
	require.NoError(t, err)
	require.Len(t, pages, 2)

	assert.Equal(t, "# 人员\n姓名: 张三, 部门: 研发\n姓名: 李四, 部门: 研发", pages[0],
		"first row becomes column names and is not a data record; the B2:B3 merge fills B3 from its top-left value")

	assert.Equal(t, "# 配置\n键: debug, 值: true", pages[1])
}

func TestParseXlsxHeaderlessFallsBackToLetters(t *testing.T) {
	pages, err := parseXlsx(buildXlsx(t), false)
	require.NoError(t, err)
	require.Len(t, pages, 2)
	assert.Equal(t, "# 人员\nA: 姓名, B: 部门\nA: 张三, B: 研发\nA: 李四, B: 研发", pages[0])
	assert.Equal(t, "# 配置\nA: 键, B: 值\nA: debug, B: true", pages[1])
}

func TestParseXlsxRejectsNonZip(t *testing.T) {
	p := writeMagicFile(t, "fake.xlsx", oleMagic)
	_, err := parseXlsx(p, true)
	require.Error(t, err)
}

// ---- page cleanup ----

func TestStripRepeatingHeaderFooter(t *testing.T) {
	pages := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		pages = append(pages, strings.Join([]string{
			"ACME Confidential Report", // header, every page
			"body line for page",
			"Page " + string(rune('1'+i)), // footer, digits vary per page
		}, "\n"))
	}
	out := stripRepeatingHeaderFooter(pages)
	for _, p := range out {
		assert.NotContains(t, p, "ACME Confidential Report", "repeating header must be stripped")
		assert.NotContains(t, p, "Page ", "page-number footer must be stripped (digits normalize)")
		assert.Contains(t, p, "body line for page", "content must survive")
	}
}

func TestStripKeepsShortDocsWithoutEvidence(t *testing.T) {
	pages := []string{"Same Co", "Same Co", "Same Co"} // too few pages
	out := stripRepeatingHeaderFooter(pages)
	assert.Equal(t, pages, out, "below the evidence floor nothing is stripped")
}

func TestStripKeepsContentLinesAtEdges(t *testing.T) {
	// first/last lines differ per page — they are content, not furniture
	pages := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		pages = append(pages, "unique opening "+string(rune('a'+i))+"\nbody\nunique ending "+string(rune('A'+i)))
	}
	out := stripRepeatingHeaderFooter(pages)
	assert.Equal(t, pages, out)
}

func TestDetectGarbledPages(t *testing.T) {
	clean := strings.Repeat("正常的中文内容", 40)
	garbled := strings.Repeat("ab\xff\xfe", 100) // decodes with U+FFFD runes
	pages := []string{clean, garbled, clean}
	assert.Equal(t, []int{2}, detectGarbledPages(pages))
	assert.Empty(t, detectGarbledPages([]string{clean, clean}))
	assert.Empty(t, detectGarbledPages([]string{"", ""}))
}

// Tika XHTML pages now render with structure markers (W2 route): the page
// selection's headings and tables survive into the chunk text.
func TestAppendPageKeepsStructureMarkers(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html><body><div class="page">
		<h1>年度报告</h1><p>总体情况良好。</p>
		<table><tr><th>指标</th><th>数值</th></tr><tr><td>营收</td><td>100</td></tr></table>
	</div></body></html>`))
	require.NoError(t, err)

	p := &DocumentTextAttachmentExtractionProcessor{config: &DocumentConfig{ChunkSize: 100}}
	var pages []string
	p.appendPage(doc.Find("div.page"), 1, nil, nil, nil, &pages)
	require.Len(t, pages, 1)
	assert.Contains(t, pages[0], "# 年度报告", "h1 survives as a markdown heading")
	assert.Contains(t, pages[0], "指标 | 数值")
	assert.Contains(t, pages[0], "营收 | 100", "table rows stay whole")
	assert.Contains(t, pages[0], "总体情况良好。")
}
