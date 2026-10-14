/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package text_attachment_extraction

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"strings"
)

// Native xlsx parsing (W2 L0, zero dependencies — the pptx.go precedent):
// Tika flattens spreadsheets into positional soup; the structured route
// emits one page per sheet with every row as `列名: 值` pairs, so chunking
// keeps whole rows together and retrieval can hit a single record. Merged
// cells are filled from their top-left value first (openpyxl-style: only
// the top-left cell stores the value); image formulas (=DISPIMG,
// _xlfn.IMAGE) carry no usable text and are dropped.

const (
	xlsxMaxRows    = 10000
	xlsxMaxSheets  = 100
	xlsxCellImgPfx = "=DISPIMG"
	xlsxCellImgAlt = "_xlfn"
)

// parseXlsx reads a workbook into one text page per sheet:
//
//	# SheetName
//	姓名: 张三, 部门: 研发
//	姓名: 李四, 部门: 测试
func parseXlsx(path string, firstRowAsHeader bool) ([]string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("not a readable xlsx (zip): %w", err)
	}
	defer zr.Close()

	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}

	shared, err := readSharedStrings(files["xl/sharedStrings.xml"])
	if err != nil {
		return nil, err
	}

	sheets, err := sheetTargets(files)
	if err != nil {
		return nil, err
	}

	pages := make([]string, 0, len(sheets))
	for i, s := range sheets {
		if i >= xlsxMaxSheets {
			break
		}
		f := files["xl/"+s.target]
		if f == nil {
			continue
		}
		page, err := sheetToPage(f, s.name, shared, firstRowAsHeader)
		if err != nil {
			return nil, err
		}
		if page != "" {
			pages = append(pages, page)
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("workbook has no readable sheets")
	}
	return pages, nil
}

// ---- xl/workbook.xml + rels: sheet order and their worksheet targets ----

type xWorkbook struct {
	Sheets struct {
		Sheet []struct {
			Name  string `xml:"name,attr"`
			RID   string `xml:"id,attr"`
			RIDNS string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheet"`
	} `xml:"sheets"`
}

type xRels struct {
	Relationship []struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	} `xml:"Relationship"`
}

type sheetRef struct{ name, target string }

func sheetTargets(files map[string]*zip.File) ([]sheetRef, error) {
	wb := xWorkbook{}
	if err := decodeXMLFile(files["xl/workbook.xml"], &wb); err != nil {
		return nil, err
	}
	rels := xRels{}
	if err := decodeXMLFile(files["xl/_rels/workbook.xml.rels"], &rels); err != nil {
		return nil, err
	}
	byID := map[string]string{}
	for _, r := range rels.Relationship {
		byID[r.ID] = strings.TrimPrefix(r.Target, "/xl/")
	}
	out := make([]sheetRef, 0, len(wb.Sheets.Sheet))
	for _, s := range wb.Sheets.Sheet {
		rid := s.RID
		if rid == "" {
			rid = s.RIDNS // real workbooks use the r: namespace prefix
		}
		target := byID[rid]
		if target == "" || !strings.Contains(target, "worksheets/") {
			continue // chartsheets etc. carry no grid
		}
		out = append(out, sheetRef{name: s.Name, target: target})
	}
	return out, nil
}

// ---- xl/sharedStrings.xml ----

type xSST struct {
	SI []struct {
		T string `xml:"t"`
	} `xml:"si"`
}

func readSharedStrings(f *zip.File) ([]string, error) {
	if f == nil {
		return nil, nil
	}
	sst := xSST{}
	if err := decodeXMLFile(f, &sst); err != nil {
		return nil, err
	}
	out := make([]string, len(sst.SI))
	for i, si := range sst.SI {
		out[i] = si.T
	}
	return out, nil
}

// ---- one worksheet: rows -> grid -> merged-cell fill -> text page ----

type xWorksheet struct {
	SheetData struct {
		Row []struct {
			R int `xml:"r,attr"`
			C []struct {
				R  string `xml:"r,attr"`
				T  string `xml:"t,attr"`
				V  string `xml:"v"`
				IS struct {
					T string `xml:"t"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"row"`
	} `xml:"sheetData"`
	MergeCells struct {
		MergeCell []struct {
			Ref string `xml:"ref,attr"`
		} `xml:"mergeCell"`
	} `xml:"mergeCells"`
}

func sheetToPage(f *zip.File, sheetName string, shared []string, firstRowAsHeader bool) (string, error) {
	ws := xWorksheet{}
	if err := decodeXMLFile(f, &ws); err != nil {
		return "", err
	}
	if len(ws.SheetData.Row) == 0 {
		return "", nil
	}

	// grid[row][col] = value, 1-based rows/cols straight from the refs
	grid := map[int]map[int]string{}
	for _, row := range ws.SheetData.Row {
		r := row.R
		if r == 0 {
			continue
		}
		for _, c := range row.C {
			col, ok := colIndexFromRef(c.R)
			if !ok {
				continue
			}
			val := cellValue(c, shared)
			if val == "" {
				continue
			}
			if grid[r] == nil {
				grid[r] = map[int]string{}
			}
			grid[r][col] = val
		}
	}

	// merged cells: every covered cell inherits the top-left value
	for _, mc := range ws.MergeCells.MergeCell {
		top, bottom, ok := parseRange(mc.Ref)
		if !ok {
			continue
		}
		v := grid[top.r][top.c]
		if v == "" {
			continue
		}
		for r := top.r; r <= bottom.r; r++ {
			for c := top.c; c <= bottom.c; c++ {
				if grid[r] == nil {
					grid[r] = map[int]string{}
				}
				grid[r][c] = v
			}
		}
	}

	minRow, maxRow, maxCol := bounds(grid)
	if maxRow == 0 {
		return "", nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n", sheetName)
	headerRow := 0
	if firstRowAsHeader && minRow <= 1 {
		headerRow = 1 // consumed as column names, not emitted as a record
	}
	for r := minRow; r <= maxRow && r-minRow < xlsxMaxRows; r++ {
		if r == headerRow {
			continue
		}
		cells := grid[r]
		if len(cells) == 0 {
			continue
		}
		var row strings.Builder
		for c := 1; c <= maxCol; c++ {
			v := cells[c]
			if v == "" {
				continue
			}
			if row.Len() > 0 {
				row.WriteString(", ")
			}
			fmt.Fprintf(&row, "%s: %s", headerName(grid, firstRowAsHeader, c), v)
		}
		if row.Len() > 0 {
			b.WriteString(row.String())
			b.WriteByte('\n')
		}
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func headerName(grid map[int]map[int]string, firstRowAsHeader bool, col int) string {
	if firstRowAsHeader {
		if v := grid[1][col]; v != "" {
			return v
		}
	}
	return colLetters(col)
}

func cellValue(c struct {
	R  string `xml:"r,attr"`
	T  string `xml:"t,attr"`
	V  string `xml:"v"`
	IS struct {
		T string `xml:"t"`
	} `xml:"is"`
}, shared []string) string {
	v := c.V
	switch c.T {
	case "s": // shared string index
		idx := 0
		if _, err := fmt.Sscanf(v, "%d", &idx); err == nil && idx >= 0 && idx < len(shared) {
			return shared[idx]
		}
		return ""
	case "inlineStr":
		return strings.TrimSpace(c.IS.T)
	case "str": // cached formula result
		if strings.Contains(v, xlsxCellImgPfx) || strings.Contains(v, xlsxCellImgAlt) {
			return ""
		}
		return v
	default:
		return v
	}
}

// colIndexFromRef maps a cell ref like "BC12" to its 1-based column index.
func colIndexFromRef(ref string) (int, bool) {
	i := 0
	for i < len(ref) {
		c := ref[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			i++
			continue
		}
		break
	}
	if i == 0 || i == len(ref) {
		return 0, false
	}
	n := 0
	for _, c := range ref[:i] {
		lower := c
		if c >= 'A' && c <= 'Z' {
			lower = c + ('a' - 'A')
		}
		n = n*26 + int(lower-'a') + 1
	}
	return n, true
}

// colLetters is the inverse of colIndexFromRef (1 -> "A", 27 -> "AA").
func colLetters(n int) string {
	if n <= 0 {
		return "?"
	}
	var b []byte
	for n > 0 {
		n--
		b = append([]byte{byte('A' + n%26)}, b...)
		n /= 26
	}
	return string(b)
}

type cellRef struct{ r, c int }

// parseRange splits "A1:B3" into its corners.
func parseRange(ref string) (cellRef, cellRef, bool) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 {
		return cellRef{}, cellRef{}, false
	}
	tl, ok1 := parseCellRef(parts[0])
	br, ok2 := parseCellRef(parts[1])
	if !ok1 || !ok2 || tl.r > br.r || tl.c > br.c {
		return cellRef{}, cellRef{}, false
	}
	return tl, br, true
}

func parseCellRef(s string) (cellRef, bool) {
	i := 0
	for i < len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			i++
			continue
		}
		break
	}
	if i == 0 {
		return cellRef{}, false
	}
	col, ok := colIndexFromRef(s[:i] + "1")
	if !ok {
		return cellRef{}, false
	}
	row := 0
	if _, err := fmt.Sscanf(s[i:], "%d", &row); err != nil || row <= 0 {
		return cellRef{}, false
	}
	return cellRef{r: row, c: col}, true
}

func bounds(grid map[int]map[int]string) (int, int, int) {
	min, max, maxCol := 0, 0, 0
	first := true
	for r, cells := range grid {
		if len(cells) == 0 {
			continue
		}
		if first || r < min {
			min = r
		}
		if r > max {
			max = r
		}
		if first {
			first = false
		}
		for c := range cells {
			if c > maxCol {
				maxCol = c
			}
		}
	}
	return min, max, maxCol
}

func decodeXMLFile(f *zip.File, v interface{}) error {
	if f == nil {
		return fmt.Errorf("xlsx part missing")
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	dec := xml.NewDecoder(rc)
	return dec.Decode(v)
}
