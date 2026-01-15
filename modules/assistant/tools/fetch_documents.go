package tools

import (
	"context"
	"strings"

	"infini.sh/coco/core"
	common2 "infini.sh/coco/modules/assistant/common"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/orm"
)

func FetchDocumentInDepth(ctx context.Context, reqMsg, replyMsg *core.ChatMessage, params *common2.RAGContext,
	docs []core.Document, inputValues map[string]any, sender core.MessageSender) error {
	if len(params.PickedDocIDS) > 0 {
		var query = orm.Query{}
		query.Conds = orm.And(orm.InStringArray("_id", params.PickedDocIDS))

		pickedFullDoc, err := fetchDocuments(&query)

		strBuilder := strings.Builder{}
		var chunkSeq = 0
		for _, v := range pickedFullDoc {
			str := "Analyzing:  " + string(v.Title) + "\n"
			strBuilder.WriteString(str)
			chunkMsg := core.NewMessageChunk(params.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.DeepRead, str, chunkSeq)
			err = sender.SendMessage(chunkMsg)
			if err != nil {
				return err
			}
		}

		detail := core.ProcessingDetails{Order: 40, Type: common.DeepRead, Description: strBuilder.String()}
		replyMsg.Details = append(replyMsg.Details, detail)

		inputValues["references"] = formatDocumentForReplyReferences(pickedFullDoc)
	}
	return nil
}
