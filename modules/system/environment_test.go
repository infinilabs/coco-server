/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	"strings"
	"testing"
)

func TestCheckBinaryProbesAgainstLocalMachine(t *testing.T) {
	libreOffice := checkBinary("libreoffice", "LibreOffice (soffice)",
		[]string{"soffice", "/Applications/LibreOffice.app/Contents/MacOS/soffice"},
		[]string{"--version"}, "impact", "hint")
	if libreOffice.Status != "ok" {
		t.Errorf("expected libreoffice ok on this machine, got %+v", libreOffice)
	}
	if !strings.Contains(libreOffice.Version, "LibreOffice") {
		t.Errorf("expected version output mentioning LibreOffice, got %q", libreOffice.Version)
	}

	pdftoppm := checkBinary("pdftoppm", "poppler (pdftoppm)", []string{"pdftoppm"}, []string{"-v"}, "impact", "hint")
	if pdftoppm.Status == "ok" && pdftoppm.Version == "" {
		t.Errorf("pdftoppm ok but no version parsed: %+v", pdftoppm)
	}
	if pdftoppm.Status != "ok" && pdftoppm.Status != "warning" {
		t.Errorf("unexpected pdftoppm status: %+v", pdftoppm)
	}

	chrome := checkBinary("chrome", "Chrome (headless)",
		[]string{
			"google-chrome", "chromium", "chromium-browser",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		},
		[]string{"--version"}, "impact", "hint")
	if chrome.Status != "ok" {
		t.Errorf("expected chrome ok on this machine, got %+v", chrome)
	}
	if chrome.Version == "" {
		t.Errorf("chrome ok but no version parsed: %+v", chrome)
	}
}

// TestCheckTika probes whatever tika endpoint the machine happens to have; it
// asserts the check is internally consistent rather than assuming the server
// is up or down.
func TestCheckTika(t *testing.T) {
	check := checkTika("http://127.0.0.1:9998")
	switch check.Status {
	case "ok":
		if check.Version == "" {
			t.Errorf("tika ok but version empty: %+v", check)
		}
		if check.Hint != "" {
			t.Errorf("tika ok should not carry an install hint: %+v", check)
		}
	case "error":
		if check.Hint == "" {
			t.Errorf("tika error should carry an install hint: %+v", check)
		}
		if check.Detail == "" {
			t.Errorf("tika error should explain why: %+v", check)
		}
	default:
		t.Errorf("unexpected tika status %q: %+v", check.Status, check)
	}
}

func TestPipelinePinsTika(t *testing.T) {
	pinned := []map[string]interface{}{
		{"document_text_attachment_extraction": map[string]interface{}{
			"tika_endpoint": "http://tika.internal:9998",
		}},
	}
	if !pipelinePinsTika(pinned) {
		t.Error("expected pinned pipeline to be detected")
	}

	blank := []map[string]interface{}{
		{"attachment_text_extraction": map[string]interface{}{
			"tika_endpoint": "  ",
		}},
	}
	if pipelinePinsTika(blank) {
		t.Error("blank override should not count as pinned")
	}

	unpinned := []map[string]interface{}{
		{"file_type_detection": map[string]interface{}{}},
		{"document_text_attachment_extraction": map[string]interface{}{
			"chunk_size": 7000,
		}},
	}
	if pipelinePinsTika(unpinned) {
		t.Error("expected unpinned pipeline to pass")
	}

	if pipelinePinsTika(nil) {
		t.Error("nil processors should not pin")
	}
}
