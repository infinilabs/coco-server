package attachment

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
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

// LoadAttachmentsForChat fetches attachment metadata for the given IDs from
// Elasticsearch using a platform-scoped ORM context. Empty IDs, non-existing
// IDs and attachments marked as deleted are silently skipped. The returned
// slice preserves the order of input ids (after filtering).
//
// This is intended for chat flows that need to read the parsed Attachment.Text
// (and related metadata) just before sending a prompt to the LLM. It must not
// be used to access KV blob payload.
func LoadAttachmentsForChat(ids []string) []*core.Attachment {
	if len(ids) == 0 {
		return nil
	}
	ormCtx := orm.NewContext()
	ormCtx.DirectReadAccess()
	ormCtx.PermissionScope(security.PermissionScopePlatform)

	out := make([]*core.Attachment, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		att := core.Attachment{}
		att.ID = id
		exists, err := orm.GetV2(ormCtx, &att)
		if err != nil {
			log.Warnf("load attachment [%s] for chat: %v", id, err)
			continue
		}
		if !exists || att.Deleted {
			continue
		}
		clone := att
		out = append(out, &clone)
	}
	return out
}

// IsAttachmentInitialParsingTerminal reports whether the attachment's
// initial_parsing stage has reached a terminal state. The second return value
// is true only when the terminal state indicates a non-success outcome
// (failed or canceled). When no stats record exists yet (e.g. just enqueued
// and the underlying ES-backed KV has not been refreshed), the attachment is
// treated as still in progress.
func IsAttachmentInitialParsingTerminal(id string) (terminal bool, failed bool) {
	stats := GetAttachmentStats(id)
	if stats == nil {
		return false, false
	}
	raw, ok := stats[core.AttachmentStageInitialParsing]
	if !ok {
		return false, false
	}
	status, _ := raw.(string)
	switch status {
	case core.StatusCompleted:
		return true, false
	case core.StatusFailed, core.StatusCanceled:
		return true, true
	default:
		return false, false
	}
}

// WaitForAttachmentsCompletion blocks until every given attachment ID reaches
// a terminal initial_parsing state (completed, failed, or canceled), the
// timeout elapses, or the parent context is canceled.
//
// Polling uses exponential backoff starting at 200ms and capped at 2s, which
// matches the eventual-consistency characteristics of the ES-backed KV store
// used for attachment_stats — callers must not assume writes by the attachment
// processor are immediately visible.
//
// failedIDs contains attachments that reached a non-success terminal state.
// On timeout the function returns context.DeadlineExceeded; on cancellation
// it returns ctx.Err(); on success it returns nil error along with any
// failedIDs collected during the wait.
//
// The optional heartbeat callback is invoked once per backoff round with the
// IDs still pending, allowing callers to emit keepalive signals on a streaming
// transport.
func WaitForAttachmentsCompletion(ctx context.Context, ids []string, timeout time.Duration, heartbeat func(pending []string)) (failedIDs []string, err error) {
	if len(ids) == 0 {
		return nil, nil
	}

	waitCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	const minDelay = 200 * time.Millisecond
	const maxDelay = 2 * time.Second

	pending := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		pending[id] = struct{}{}
	}

	delay := minDelay
	for {
		// Single sweep over currently pending IDs.
		for id := range pending {
			terminal, didFail := IsAttachmentInitialParsingTerminal(id)
			if !terminal {
				continue
			}
			if didFail {
				failedIDs = append(failedIDs, id)
			}
			delete(pending, id)
		}
		if len(pending) == 0 {
			return failedIDs, nil
		}
		if heartbeat != nil {
			still := make([]string, 0, len(pending))
			for id := range pending {
				still = append(still, id)
			}
			heartbeat(still)
		}
		select {
		case <-waitCtx.Done():
			return failedIDs, waitCtx.Err()
		case <-time.After(delay):
		}
		if delay < maxDelay {
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
}
