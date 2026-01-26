package attachment

import (
	"fmt"
	"io"
	"mime/multipart"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Helper function to upload the attachment specified by [file] to the
// blob store.
//
// Arguments:
//
//   - If [fileID] is not an empty string, it will be used as the file ID.
//     Otherwise, a random ID will be created and used.
//   - If [ownerID] is not empty, the created attached will set the owner to it.
//     Otherwise, owner information will be extracted from context [ctx].
//   - If [fileVerboseText] is not empty, it will be stored in the attachment's text
//     field (e.g., extracted text from an image).
//   - If [metadata] is not empty, it will be stored in the attachment's metadata
//     field.
//   - [replaceIfExists]: If this is true and there is already an attachment with
//     the same file ID exists, replace it.
//
// Return value:
//   - attachment ID: it will be [fileID] if it is not empty
func UploadToBlobStore(ctx *orm.Context, fileID string, file multipart.File, header *multipart.FileHeader, fileName string, ownerID string, metadata util.MapStr, fileVerboseText string, replaceIfExists bool) (string, error) {
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
	mimeType, err := DetectMimeType(fileName, file, header)

	attachment := core.Attachment{}
	attachment.ID = fileID
	attachment.Name = fileName
	attachment.Size = fileSize
	attachment.MimeType = mimeType
	attachment.Icon = getFileExtension(fileName)
	attachment.URL = fmt.Sprintf("/attachment/%v", fileID)
	attachment.Text = fileVerboseText

	if ownerID != "" {
		attachment.SetOwnerID(ownerID)
	}

	if metadata != nil {
		if attachment.Metadata == nil {
			attachment.Metadata = metadata
		} else {
			for k, v := range metadata {
				attachment.Metadata[k] = v
			}
		}
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

	log.Tracef("uploading files: attachment [%s/%s] created", fileName, fileID)

	//save attachment payload
	//
	// kv.AddValue will replace the previous value if it already exists so we
	// don't need to check [replaceIfExists] here.
	err = kv.AddValue(core.AttachmentKVBucket, []byte(fileID), data)
	if err != nil {
		panic(err)
	}
	log.Tracef("uploading files: payload [%s/%s] stored", fileName, fileID)

	log.Debugf("file [%s] successfully uploaded, size: %v", fileName, fileSize)
	return fileID, nil
}

func getAttachmentStatus(ids []string) map[string]util.MapStr {
	out := make(map[string]util.MapStr)
	for _, id := range ids {
		stats := GetAttachmentStats(id)
		if stats != nil {
			out[id] = stats
		} else {
			//TODO remove this when the actual pipeline is ready
			obj := util.MapStr{
				core.AttachmentStageInitialParsing: "completed",
			}
			out[id] = obj
		}
	}

	return out
}

func GetAttachmentStats(id string) util.MapStr {
	v, err := kv.GetValue(core.AttachmentStatsBucket, []byte(id))
	if err == nil {
		obj := util.MapStr{}
		util.MustFromJSONBytes(v, &obj)

		return obj
	}

	return nil
}

func UpdateAttachmentStats(id string, stats util.MapStr) {
	if stats == nil {
		return
	}

	prevStats := GetAttachmentStats(id)
	if prevStats == nil {
		prevStats = stats
	} else {
		for k, v := range stats {
			prevStats[k] = v
		}
	}

	err := kv.AddValue(core.AttachmentStatsBucket, []byte(id), util.MustToJSONBytes(prevStats))
	if err != nil {
		panic(err)
	}
}
