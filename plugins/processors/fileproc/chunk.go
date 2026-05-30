/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package fileproc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/attachment"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

// Extraction holds the result of extracting content from a document.
type Extraction struct {
	// Pages contains the text content of each page.
	Pages []string
	// Attachments holds the IDs of every core.Attachment created for embedded files.
	Attachments []string
}

// SplitPagesToChunks splits page texts into fixed-size character chunks and
// tracks the page range for each chunk.
func SplitPagesToChunks(pages []string, chunkSize int) []core.DocumentChunk {
	if chunkSize <= 0 {
		return nil
	}
	if len(pages) == 0 {
		return make([]core.DocumentChunk, 0)
	}

	var chunks []core.DocumentChunk
	buf := make([]rune, 0, chunkSize)
	startPage := 0
	lastPage := 0

	for idx, page := range pages {
		pageNumber := idx + 1
		pageChars := []rune(page)

		for len(pageChars) > 0 {
			nWant := chunkSize - len(buf)
			nTake := min(nWant, len(pageChars))
			buf = append(buf, pageChars[:nTake]...)

			if startPage == 0 {
				startPage = pageNumber
			}
			if len(buf) == chunkSize && lastPage == 0 {
				lastPage = pageNumber
				chunks = append(chunks, core.DocumentChunk{
					Range: core.ChunkRange{Start: startPage, End: lastPage},
					Text:  string(buf),
				})
				buf = buf[:0]
				startPage = 0
				lastPage = 0
			}

			pageChars = pageChars[nTake:]
		}
	}

	if len(buf) != 0 {
		if startPage == 0 {
			panic("unreachable: buf updated but startPage is still 0")
		}
		if lastPage == 0 {
			lastPage = len(pages)
		}
		chunks = append(chunks, core.DocumentChunk{
			Range: core.ChunkRange{Start: startPage, End: lastPage},
			Text:  string(buf),
		})
	}

	return chunks
}

// UploadAttachmentsToBlobStore uploads the files in dir as attachments for doc.
// nameToId maps filename → pre-assigned UUID; nameToText maps filename → OCR text;
// nameToPageNums maps filename → page numbers where the attachment appears.
func UploadAttachmentsToBlobStore(ctx context.Context, dir string, doc *core.Document, nameToId map[string]string, nameToText map[string]string, nameToPageNums map[string][]int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		name := entry.Name()
		if name == "__METADATA__" || name == "__TEXT__" {
			continue
		}

		fullPath := filepath.Join(dir, name)
		uploadFile, err := os.Open(fullPath)
		if err != nil {
			return fmt.Errorf("failed to open extracted file for upload %s: %w", fullPath, err)
		}

		fileID, ok := nameToId[name]
		if !ok {
			panic(fmt.Sprintf("unreachable: attachment ID not found for file %s; all files in the directory should have been pre-processed and assigned a UUID", name))
		}

		ormCtx := orm.NewContextWithParent(ctx)
		ormCtx.DirectAccess()
		ormCtx.PermissionScope(security.PermissionScopePlatform)

		ownerID := doc.GetOwnerID()
		fileContent := nameToText[name]
		pageNums := nameToPageNums[name]

		metadata := util.MapStr{}
		if doc.ID != "" {
			metadata["document_id"] = doc.ID
		}
		if len(pageNums) > 0 {
			metadata["document_page_num"] = pageNums
		}

		if _, err = attachment.UploadToBlobStore(ormCtx, fileID, uploadFile, nil, name, ownerID, metadata, fileContent, true); err != nil {
			return fmt.Errorf("failed to upload attachment %s: %w", name, err)
		}
	}
	return nil
}
