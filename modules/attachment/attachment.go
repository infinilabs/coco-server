// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package attachment

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

//	curl -X POST http://localhost:9000/chat/session_id/_upload \
//	 -H "Authorization: Bearer YOUR_TOKEN" \
//	 -F "files=@/path/to/your/file1.txt" \
//	 -F "files=@/path/to/your/file2.jpg"
func (h APIHandler) uploadAttachment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Limit upload size (e.g., 20MB total)
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		h.WriteError(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	ctx := orm.NewContextWithParent(r.Context())
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		h.WriteError(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	ctx.Refresh = orm.WaitForRefresh

	attachmentIDs := []string{}
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			h.WriteError(w, fmt.Sprintf("Failed to open file %s", fileHeader.Filename), http.StatusInternalServerError)
			return
		}
		// Upload to S3
		if fileID, err := uploadToBlobStore(ctx, file, fileHeader.Filename); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			attachmentIDs = append(attachmentIDs, fileID)
		}
	}

	result := util.MapStr{}
	result["attachments"] = attachmentIDs

	h.WriteAckJSON(w, true, 200, result)
}

type AttachmentsRequest struct {
	Attachments []string `json:"attachments"`
}

func (h APIHandler) getAttachments(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	filterReq := AttachmentsRequest{}
	body, _ := h.GetRawBody(req)
	if len(body) > 0 {
		util.MustFromJSONBytes(body, &filterReq)
	}

	var err error
	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &common.Attachment{})

	builder, err := orm.NewQueryBuilderFromRequest(req, "name", "description")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	builder.Filter(orm.MustNotQuery(orm.TermQuery("deleted", true)))

	if len(filterReq.Attachments) > 0 {

		builder.Filter(orm.TermsQuery("id", filterReq.Attachments))

	}

	docs := []common.Attachment{}
	err, res := core.SearchV2WithResultItemMapper(ctx, &docs, builder, nil)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, res.Raw)
	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) getAttachment(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	fileID := ps.MustGetParameter("file_id")
	data, err := kv.GetValue(AttachmentKVBucket, []byte(fileID))
	if err != nil || len(data) == 0 {
		panic("invalid attachment")
	}
	attachment, exists, err := h.getAttachmentMetadata(req, fileID)
	if !exists {
		h.WriteGetMissingJSON(w, fileID)
		return
	}
	if err != nil || attachment == nil {
		panic(err)
	}

	// Set headers
	w.Header().Set("Content-Disposition", "attachment; filename=\""+attachment.Name+"\"")
	if attachment.MimeType != "" {
		w.Header().Set("Content-Type", attachment.MimeType)
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))

	// Write file data to response
	_, _ = w.Write(data)
}

func (h APIHandler) getAttachmentMetadata(req *http.Request, fileID string) (*common.Attachment, bool, error) {
	attachment := common.Attachment{}
	attachment.ID = fileID
	ctx := orm.NewContextWithParent(req.Context())

	exists, err := orm.GetV2(ctx, &attachment)
	if err != nil {
		return nil, exists, err
	}

	if !exists {
		//TODO kv exists, should cleanup kv store
		_ = log.Warnf("file meta %v not exists, but kv exists", fileID)
		return nil, exists, err
	}

	if attachment.Deleted {
		_ = log.Warnf("attachment %v exists but was deleted", fileID)
		return nil, false, errors.New("attachment not found")
	}

	return &attachment, exists, nil
}

func (h APIHandler) checkAttachment(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	fileID := ps.MustGetParameter("file_id")
	attachment, exists, err := h.getAttachmentMetadata(req, fileID)
	if !exists {
		h.WriteGetMissingJSON(w, fileID)
		return
	}

	if err != nil || attachment == nil {
		panic(err)
	}

	w.Header().Set("Filename", attachment.Name)
	if attachment.Created != nil {
		w.Header().Set("Created", fmt.Sprintf("%d", attachment.Created.Unix()))
	} else {
		w.Header().Set("Created", "")
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", attachment.Size))
	w.WriteHeader(200)
}

func (h APIHandler) deleteAttachment(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	fileID := ps.MustGetParameter("file_id")
	attachment, exists, err := h.getAttachmentMetadata(req, fileID)
	if !exists {
		h.WriteGetMissingJSON(w, fileID)
		return
	}

	if err != nil || attachment == nil {
		panic(err)
	}

	attachment.Deleted = true
	t := time.Now()
	attachment.LastUpdatedBy = &common.EditorInfo{UpdatedAt: &t}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh

	err = orm.Update(ctx, attachment)
	if err != nil {
		panic(err)
	}

	h.WriteDeletedOKJSON(w, fileID)
}

const AttachmentKVBucket = "file_attachments"

func getFileExtension(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if len(ext) > 0 {
		return ext[1:] // remove the dot
	}
	return ""
}

func getMimeType(file multipart.File) (string, error) {
	// Read first 512 bytes for MIME type detection
	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil {
		return "", err
	}

	// Reset the file pointer after reading
	_, err = file.Seek(0, 0)
	if err != nil {
		return "", err
	}

	// Detect content type
	mimeType := http.DetectContentType(buffer)
	return mimeType, nil
}

func uploadToBlobStore(ctx *orm.Context, file multipart.File, fileName string) (string, error) {
	defer func() {
		_ = file.Close()
	}()

	// Read file content into memory
	data, err := io.ReadAll(file)
	if err != nil || len(data) == 0 {
		return "", fmt.Errorf("failed to read file %s: %v", fileName, err)
	}

	fileID := util.GetUUID()
	fileSize := len(data)
	mimeType, _ := getMimeType(file)

	attachment := common.Attachment{}
	attachment.ID = fileID
	attachment.Name = fileName
	attachment.Size = fileSize
	attachment.MimeType = mimeType
	attachment.Icon = getFileExtension(fileName)
	attachment.URL = fmt.Sprintf("/attachment/%v", fileID)
	//attachment.Owner //TODO

	//save attachment metadata
	err = orm.Create(ctx, &attachment)
	if err != nil {
		panic(err)
	}

	//save attachment payload
	err = kv.AddValue(AttachmentKVBucket, []byte(fileID), data)
	if err != nil {
		panic(err)
	}

	log.Debugf("file [%s] successfully uploaded, size: %v", fileName, fileSize)
	return fileID, nil
}
