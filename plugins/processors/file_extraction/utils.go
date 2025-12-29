package file_extraction

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/attachment"
	"infini.sh/framework/core/orm"
)

func tikaGetTextPlain(tikaRequestCtx context.Context, tikaEndpoint string, timeout int, path string) (io.ReadCloser, error) {
	if tikaEndpoint == "" || path == "" {
		return nil, fmt.Errorf("[tika_endpoint] and [path] should not be empty")
	}

	// 1. Open the file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	// Note: file is consumed by the HTTP request body and closed by the HTTP client

	// 2. Create the HTTP Request
	// Tika expects a PUT request with the file binary as the body.
	url := tikaEndpoint + "/tika"
	req, err := http.NewRequestWithContext(tikaRequestCtx, "PUT", url, file)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("X-Tika-PDFextractInlineImages", "true")

	// 4. Send the Request
	// We use a client with a generous timeout because OCR is slow.
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Tika: %w", err)
	}

	// 5. Check Status Code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("Closing response body returned error: %s", err)
		}
		return nil, fmt.Errorf("tika returned error %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

func tikaGetTextHtml(tikaRequestCtx context.Context, tikaEndpoint string, timeout int, path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file [%s]: %w", path, err)
	}
	// Note: file is consumed by the HTTP request body and closed by the HTTP client

	url := fmt.Sprintf("%s/tika", tikaEndpoint)
	req, err := http.NewRequestWithContext(tikaRequestCtx, "PUT", url, file)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", path, err)
	}
	req.Header.Set("Accept", "text/html")
	// If [path] points to a PDF file, we need this flag to let it return inline images.
	req.Header.Set("X-Tika-PDFextractInlineImages", "true")

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request tika for %s: %w", path, err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("Closing response body returned error: %s", err)
		}
		return nil, fmt.Errorf("tika returned status %d for %s: %s", resp.StatusCode, path, string(body))
	}

	return resp.Body, nil
}

// Let Tika unpack all the attachments of the file specified by [filePath]
// to the directory pointed by [to].
func tikaUnpackAllTo(tikaRequestCtx context.Context, tikaEndpoint, filePath, to string, timeout int) error {
	// 1. Open the source file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", filePath, err)
	}
	// Note: file is consumed by the HTTP request body and closed by the HTTP client

	// 2. Construct Tika URL (ensure no double slashes)
	endpoint := strings.TrimRight(tikaEndpoint, "/")
	unpackUrl := fmt.Sprintf("%s/unpack/all", endpoint)

	// 3. Create the Request
	req, err := http.NewRequestWithContext(tikaRequestCtx, "PUT", unpackUrl, file)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Suggest Tika to extract inline images from PDFs (OCR trigger)
	req.Header.Set("X-Tika-PDFextractInlineImages", "true")
	// Tika usually detects content type, but explicit headers like Accept are good practice
	req.Header.Set("Accept", "application/zip")

	// 4. Send Request
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to communicate with tika: %w", err)
	}
	defer DeferClose(resp.Body)

	if resp.StatusCode != http.StatusOK {
		// Read a bit of the body for debugging
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("tika unpack failed with status %d: %s", resp.StatusCode, string(snippet))
	}

	// 5. Save the Response (ZIP stream) to a Temporary File.
	// We MUST save to disk or memory because archive/zip requires io.ReaderAt (random access),
	// but HTTP response is a stream. Disk is safer for large files.
	tmpFile, err := os.CreateTemp("", "tika-unpack-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp zip file: %w", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name()) // Cleanup temp zip after extraction
	}()

	// Stream Tika response -> Temp File
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save tika response to disk: %w", err)
	}

	// 6. Unzip the Temp File to the destination
	// We need to re-open or seek because the pointer is at the end after io.Copy
	// Simply opening by name is easiest here since we have the path.
	zipReader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open extracted zip: %w", err)
	}
	defer DeferClose(zipReader)

	if err := os.MkdirAll(to, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", to, err)
	}

	// Extract files
	for _, f := range zipReader.File {
		err := extractZipFile(f, to)
		if err != nil {
			return fmt.Errorf("failed to extract file %s: %w", f.Name, err)
		}
	}

	return nil
}

// Directory [dir] contains document [doc]'s attachments, upload them to
// the blob store.
func uploadAttachmentsToBlobStore(ctx context.Context, dir string, doc *core.Document) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}

	for _, entry := range entries {
		// We only process regular file
		if !entry.Type().IsRegular() {
			continue
		}

		// We only want attachments, not text and file metadata
		if entry.Name() == "__METADATA__" || entry.Name() == "__TEXT__" {
			continue
		}

		// Process image/attachment
		fullPath := filepath.Join(dir, entry.Name())

		uploadFile, err := os.Open(fullPath)
		if err != nil {
			return fmt.Errorf("failed to open extracted file for upload %s: %w", fullPath, err)
		}
		// uploadFile will be closed in `attachment.UploadToBlobStore`

		fileID := doc.ID + entry.Name()

		ormCtx := orm.NewContextWithParent(ctx)
		// Grant read/write access to the database, which is needed because this
		// is a background processor, which has no user token stored in ctx.
		ormCtx.DirectAccess()
		ownerID := doc.GetOwnerID()

		_, err = attachment.UploadToBlobStore(ormCtx, fileID, uploadFile, entry.Name(), ownerID, true)
		if err != nil {
			return fmt.Errorf("failed to upload attachment %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// Splits page texts into chunks using character count as a token proxy
// and tracks the page range for each chunk.
func SplitPagesToChunks(pages []string, chunkSize int) []core.DocumentChunk {
	// Early return
	if chunkSize <= 0 {
		return nil
	}
	if len(pages) == 0 {
		return make([]core.DocumentChunk, 0)
	}

	var chunks []core.DocumentChunk

	buf := make([]rune, 0, chunkSize)
	// Value 0 means `startPage` and `lastPage` are not initialized
	startPage := 0
	lastPage := 0

	for idx, page := range pages {
		pageNumber := idx + 1
		pageChars := []rune(page)

		for len(pageChars) > 0 {
			nCharsWeWant := chunkSize - len(buf)
			nCharsWeCanTake := min(nCharsWeWant, len(pageChars))
			chars := pageChars[:nCharsWeCanTake]
			buf = append(buf, chars...)

			// Update page range after modifying `buf`
			if startPage == 0 {
				startPage = pageNumber
			}
			if len(buf) == chunkSize && lastPage == 0 {
				lastPage = pageNumber

				// `buf` is ready
				textChunk := string(buf)
				chunkRange := core.ChunkRange{
					Start: startPage,
					End:   lastPage,
				}

				chunks = append(chunks, core.DocumentChunk{
					Range: chunkRange,
					Text:  textChunk,
					// this field remains uninitialized
					// Embedding: core.Embedding{},
				})

				// clear buf and states
				buf = buf[:0]
				startPage = 0
				lastPage = 0
			}

			// Remove the consumed bytes from `pageChars`
			pageChars = pageChars[nCharsWeCanTake:]
		}
	}

	// We may have a chunk whose size is smaller than `chunkSize`
	if len(buf) != 0 {
		// startPage should be updated
		if startPage == 0 {
			panic("unreachable: buf got updated but startPage is still 0")
		}
		// Set lastPage
		if lastPage == 0 {
			lastPage = len(pages)
		}

		// `buf` is ready
		textChunk := string(buf)
		chunkRange := core.ChunkRange{
			Start: startPage,
			End:   lastPage,
		}
		chunks = append(chunks, core.DocumentChunk{
			Range: chunkRange,
			Text:  textChunk,
			// this field remains uninitialized
			// Embedding: core.Embedding{},
		})
	}

	return chunks
}

// extractZipFile handles the individual file extraction logic with security checks
//
// It is a helper function of [tikaUnpackAllTo()].
func extractZipFile(f *zip.File, dest string) error {
	// 1. "Zip Slip" Vulnerability Protection
	// Ensure the calculated path is actually inside the destination directory
	fpath := filepath.Join(dest, f.Name)
	if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path: %s", fpath)
	}

	// 2. Handle Directories
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	// 3. Create Parent Directory (if it doesn't exist)
	if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
		return err
	}

	// 4. Copy Content
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

// DeferClose is a utility function used to check the return from
// Close in a defer statement.
func DeferClose(c io.Closer) {
	closeErr := c.Close()
	if closeErr != nil {
		log.Errorf("Close() failed with error: %s", closeErr)
	}
}
