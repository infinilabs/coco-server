/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package fileproc

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TikaGetTextPlain sends [path] to Tika and returns a text/plain reader.
func TikaGetTextPlain(ctx context.Context, tikaEndpoint string, timeout int, path string) (io.ReadCloser, error) {
	if tikaEndpoint == "" || path == "" {
		return nil, fmt.Errorf("[tika_endpoint] and [path] should not be empty")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	url := tikaEndpoint + "/tika"
	req, err := http.NewRequestWithContext(ctx, "PUT", url, file)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("X-Tika-PDFextractInlineImages", "true")

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to send request to Tika: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("tika returned error %d: %s", resp.StatusCode, string(body))
	}
	return resp.Body, nil
}

// OCR performs OCR on the file at path using Tika and returns cleaned text.
func OCR(ctx context.Context, tikaEndpoint string, timeout int, path string) (string, error) {
	rc, err := TikaGetTextPlain(ctx, tikaEndpoint, timeout, path)
	if err != nil {
		return "", err
	}
	defer DeferClose(rc)

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	text := string(data)

	// Remove control characters
	text = strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, text)

	// Collapse multiple spaces
	re := regexp.MustCompile(` +`)
	text = re.ReplaceAllString(text, " ")
	return strings.TrimSpace(text), nil
}

// TikaGetTextHtml sends [path] to Tika and returns a text/html reader.
func TikaGetTextHtml(ctx context.Context, tikaEndpoint string, timeout int, path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file [%s]: %w", path, err)
	}

	url := fmt.Sprintf("%s/tika", tikaEndpoint)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, file)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to create request for %s: %w", path, err)
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("X-Tika-PDFextractInlineImages", "true")

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to request tika for %s: %w", path, err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("tika returned status %d for %s: %s", resp.StatusCode, path, string(body))
	}
	return resp.Body, nil
}

// TikaUnpackAllTo unpacks all attachments of filePath to the directory to.
func TikaUnpackAllTo(ctx context.Context, tikaEndpoint, filePath, to string, timeout int) error {
	tikaCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", filePath, err)
	}

	endpoint := strings.TrimRight(tikaEndpoint, "/")
	unpackURL := fmt.Sprintf("%s/unpack/all", endpoint)

	req, err := http.NewRequestWithContext(tikaCtx, "PUT", unpackURL, file)
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Tika-PDFextractInlineImages", "true")
	req.Header.Set("Accept", "application/zip")

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to communicate with tika: %w", err)
	}
	defer DeferClose(resp.Body)

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("tika unpack failed with status %d: %s", resp.StatusCode, string(snippet))
	}

	tmpFile, err := os.CreateTemp("", "tika-unpack-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp zip file: %w", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save tika response to disk: %w", err)
	}

	zipReader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open extracted zip: %w", err)
	}
	defer DeferClose(zipReader)

	if err := os.MkdirAll(to, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", to, err)
	}

	for _, f := range zipReader.File {
		if err := extractZipFile(f, to); err != nil {
			return fmt.Errorf("failed to extract file %s: %w", f.Name, err)
		}
	}
	return nil
}

// extractZipFile extracts a single zip entry to dest with Zip Slip protection.
func extractZipFile(f *zip.File, dest string) error {
	fpath := filepath.Join(dest, f.Name)
	if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path: %s", fpath)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(fpath, os.ModePerm)
	}

	if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
		return err
	}

	outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer DeferClose(outFile)

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer DeferClose(rc)

	_, err = io.Copy(outFile, rc)
	return err
}

// GetSortedSlideFiles finds and sorts slide XMLs in a PPTX zip (natural order).
func GetSortedSlideFiles(r *zip.ReadCloser) ([]*zip.File, error) {
	var slides []*zip.File
	re := regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)

	for _, f := range r.File {
		if re.MatchString(f.Name) {
			slides = append(slides, f)
		}
	}

	if len(slides) == 0 {
		return nil, fmt.Errorf("no slides found")
	}

	sort.Slice(slides, func(i, j int) bool {
		numI, _ := strconv.Atoi(re.FindStringSubmatch(slides[i].Name)[1])
		numJ, _ := strconv.Atoi(re.FindStringSubmatch(slides[j].Name)[1])
		return numI < numJ
	})
	return slides, nil
}

// GetSlideRelationships parses the .rels file for slidePath and returns a map
// of relationship ID → image filename.
func GetSlideRelationships(r *zip.ReadCloser, slidePath string) (map[string]string, error) {
	dir := filepath.Dir(slidePath)
	base := filepath.Base(slidePath)
	relsPath := strings.ReplaceAll(filepath.Join(dir, "_rels", base+".rels"), "\\", "/")

	relsMap := make(map[string]string)

	var relFile *zip.File
	for _, f := range r.File {
		if f.Name == relsPath {
			relFile = f
			break
		}
	}
	if relFile == nil {
		return relsMap, nil
	}

	rc, err := relFile.Open()
	if err != nil {
		return nil, err
	}
	defer DeferClose(rc)

	type Relationship struct {
		Id     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	}
	type Relationships struct {
		List []Relationship `xml:"Relationship"`
	}

	var rels Relationships
	if err := xml.NewDecoder(rc).Decode(&rels); err != nil {
		return nil, err
	}

	for _, rel := range rels.List {
		if IsImage(rel.Target) {
			relsMap[rel.Id] = filepath.Base(rel.Target)
		}
	}
	return relsMap, nil
}
