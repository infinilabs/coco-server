/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

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

	"infini.sh/coco/core"
	api1 "infini.sh/framework/core/api"
	"infini.sh/framework/core/elastic"

	log "github.com/cihub/seelog"
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
		if fileID, err := UploadToBlobStore(ctx, "", file, fileHeader.Filename, "", "", nil, "", false); err != nil {
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
	orm.WithModel(ctx, &core.Attachment{})

	builder, err := orm.NewQueryBuilderFromRequest(req, "name", "description")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(builder.Sorts()) == 0 {
		builder.SortBy(orm.Sort{Field: "created", SortType: orm.DESC})
	}

	builder.Filter(orm.MustNotQuery(orm.TermQuery("deleted", true)))

	if len(filterReq.Attachments) > 0 {

		builder.Filter(orm.TermsQuery("id", filterReq.Attachments))

	}

	docs := []core.Attachment{}
	err, res := elastic.SearchV2WithResultItemMapper(ctx, &docs, builder, nil)
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
	data, err := kv.GetValue(core.AttachmentKVBucket, []byte(fileID))
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

func (h APIHandler) getAttachmentMetadata(req *http.Request, fileID string) (*core.Attachment, bool, error) {
	attachment := core.Attachment{}
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
	attachment.LastUpdatedBy = &core.EditorInfo{UpdatedAt: &t}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh

	err = orm.Update(ctx, attachment)
	if err != nil {
		panic(err)
	}

	h.WriteDeletedOKJSON(w, fileID)
}

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

// Helper function to upload the attachment specified by [file] to the
// blob store.
//
// Arguments:
//
//   - If [fileID] is not an empty string, it will be used as the file ID.
//     Otherwise, a random ID will be created and used.
//   - If [ownerID] is not empty, the created attached will set the owner to it.
//     Otherwise, owner information will be extracted from cotnext [ctx].
//   - If [documentID] is not empty, it indicates this attachment belongs to a
//     document, and the ID will be stored in the attachment's metadata.
//   - If [documentPageNums] is not empty, it indicates the page numbers where
//     this attachment appears in the document.
//   - If [fileContent] is not empty, it will be stored in the attachment's text
//     field (e.g., extracted text from an image).
//   - [replaceIfExists]: If this is true and there is already an attachment with
//     the same file ID eixsts, replace it.
//
// Return value:
//   - attachment ID: it will be [fileID] if it is not empty
func UploadToBlobStore(ctx *orm.Context, fileID string, file multipart.File, fileName string, ownerID string, documentID string, documentPageNums []int, fileContent string, replaceIfExists bool) (string, error) {
	defer func() {
		_ = file.Close()
	}()

	// Read file content into memory
	data, err := io.ReadAll(file)
	if err != nil || len(data) == 0 {
		return "", fmt.Errorf("failed to read file %s: %v", fileName, err)
	}

	if fileID == "" {
		fileID = util.GetUUID()
	}
	fileSize := len(data)
	mimeType, _ := getMimeType(file)

	attachment := core.Attachment{}
	attachment.ID = fileID
	attachment.Name = fileName
	attachment.Size = fileSize
	attachment.MimeType = mimeType
	attachment.Icon = getFileExtension(fileName)
	attachment.URL = fmt.Sprintf("/attachment/%v", fileID)
	attachment.Text = fileContent

	if ownerID != "" {
		attachment.SetOwnerID(ownerID)
	}

	if documentID != "" {
		if attachment.Metadata == nil {
			attachment.Metadata = make(map[string]interface{})
		}
		attachment.Metadata["document_id"] = documentID
	}

	if len(documentPageNums) > 0 {
		if attachment.Metadata == nil {
			attachment.Metadata = make(map[string]interface{})
		}
		attachment.Metadata["document_page_num"] = documentPageNums
	}

	//save attachment metadata
	if replaceIfExists {
		err = orm.Upsert(ctx, &attachment)
	} else {
		err = orm.Create(ctx, &attachment)
	}
	if err != nil {
		panic(err)
	}

	//save attachment payload
	//
	// kv.AddValue will replace the previous value if it already exists so we
	// don't need to check [replaceIfExists] here.
	err = kv.AddValue(core.AttachmentKVBucket, []byte(fileID), data)
	if err != nil {
		panic(err)
	}

	log.Debugf("file [%s] successfully uploaded, size: %v", fileName, fileSize)
	return fileID, nil
}

func (h APIHandler) getAttachmentStats(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("file_id")
	out := getAttachmentStats([]string{id})

	if out != nil {
		o, ok := out[id]
		if ok {
			api1.WriteJSON(w, o, 200)
			return
		}
	}

	api1.WriteJSON(w, util.MapStr{}, 200)
}

type AttachmentStatsRequest struct {
	Attachments []string `json:"attachments"`
}

func (h APIHandler) batchGetAttachmentStats(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	reqObj := AttachmentStatsRequest{}
	api1.MustDecodeJSON(req, &reqObj)
	if len(reqObj.Attachments) == 0 {
		api1.WriteJSON(w, req, 200)
		return
	}

	output := map[string]util.MapStr{}
	for _, id := range reqObj.Attachments {
		output[id] = util.MapStr{
			"initial_parsing": "completed",
		}
	}
	api1.WriteJSON(w, output, 200)

}
