/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package text_attachment_extraction

import (
	"os"
)

// Magic-number extension correction (W2 L0): filenames lie — WPS saves,
// manual renames and "final_v2.docx actually .doc" are routine. The
// extension-based routing below only misbehaves for the native zip/xml
// parsers (pptx, xlsx): an OLE payload breaks them, and a zip payload
// renamed .ppt deserves the native path. Tika (the default route)
// content-sniffs by itself and needs no correction.
//
// The correction is deliberately conservative: it only rewrites extensions
// where the native parser would otherwise fail, and records every mismatch
// on the document for the processing timeline (W13a).

var (
	oleMagic = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	zipMagic = []byte{'P', 'K', 0x03, 0x04}
)

// effectiveExt sniffs the file's leading bytes and corrects the extension
// when the bytes contradict it for a route that cares. Returns the ext to
// route on and a human-readable mismatch note ("" when consistent).
func effectiveExt(localPath, ext string) (string, string) {
	f, err := os.Open(localPath)
	if err != nil {
		return ext, "" // unreadable: keep the extension, the route will surface the error
	}
	defer f.Close()
	head := make([]byte, 8)
	n, _ := f.Read(head)
	head = head[:n]

	isOLE := len(head) >= len(oleMagic) && string(head[:len(oleMagic)]) == string(oleMagic)
	isZip := len(head) >= len(zipMagic) && string(head[:len(zipMagic)]) == string(zipMagic)

	switch {
	case isOLE && (ext == ".pptx" || ext == ".pptm"):
		// legacy binary PowerPoint wearing an OOXML name — the native
		// pptx (zip) parser cannot read it; Tika can
		return ".ppt-ole", ext + " is actually legacy OLE PowerPoint"
	case isOLE && ext == ".xlsx":
		return ".xls", ext + " is actually legacy OLE Excel"
	case isOLE && ext == ".docx":
		return ".doc", ext + " is actually legacy OLE Word"
	case isZip && ext == ".ppt":
		return ".pptx", ext + " is actually an OOXML (zip) presentation"
	case isZip && ext == ".xls":
		return ".xlsx", ext + " is actually an OOXML (zip) workbook"
	case isZip && ext == ".doc":
		return ".docx", ext + " is actually an OOXML (zip) document"
	}
	return ext, ""
}

// mimeMismatchTag is the metadata key recording a magic-number correction.
const mimeMismatchTag = "magic_ext_corrected"
