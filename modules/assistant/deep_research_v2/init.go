package deep_research

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

func RunDeepResearchV2(ctx context.Context, query string, config *core.DeepResearchConfig, reqMsg, replyMsg *core.ChatMessage, sender core.MessageSender) error {

	//response
	reasoningBuffer := strings.Builder{}
	messageBuffer := strings.Builder{}
	// note: we use defer to ensure that the response message is saved after processing
	// even if user cancels the task or if an error occurs
	defer func() {
		//save response message to system
		if messageBuffer.Len() > 0 {
			log.Trace("update reply message")
			replyMsg.Message = messageBuffer.String()
		} else {
			log.Warnf("seems empty reply for query: %v", replyMsg)
		}
		if reasoningBuffer.Len() > 0 {
			detail := core.ProcessingDetails{Order: 50, Type: common.Think, Description: reasoningBuffer.String()}
			replyMsg.Details = append(replyMsg.Details, detail)
		}
	}()

	log.Infof("正在启动 Deep-Research 研究代理，查询内容：%s\n", query)

	graph, err := NewGraph()
	if err != nil {
		panic(errors.Errorf("Failed to create graph: %v", err))
	}

	initialState := &State{
		Config: config,
		Sender: sender,
		Request: Request{
			Query: query,
		},
	}

	result, err := graph.Invoke(ctx, initialState)
	if err != nil {
		panic(errors.Errorf("Graph execution failed: %v", err))
	}

	finalState := result.(*State)
	log.Info("\n=== Final Report ===")
	log.Info(finalState.MarkdownReport)

	attachment := saveReport(ctx, "Research-Report.md", finalState.MarkdownReport)

	report := util.MapStr{}
	report["title"] = attachment.Name
	report["url"] = attachment.URL
	report["created"] = attachment.Created
	report["attachment"] = attachment.ID
	finalState.Sender.SendChunkMessage(core.MessageTypeAssistant, common.ResearchReporterEnd, util.MustToJSON(report), 0)

	log.Info("报告生成完成：")
	log.Info("  MarkdownReport 长度: ", len(finalState.MarkdownReport))
	log.Info("  FinalReport 长度: ", len(finalState.FinalReport))
	log.Info("  章节大纲数量: ", len(finalState.ChapterOutline))
	log.Info("  章节大纲: ", util.ToJson(finalState.ChapterOutline, true))
	log.Info("  ChapterContents 数量: ", len(finalState.ChapterContents))
	log.Info("  ChapterContents: ", util.ToJson(finalState.ChapterContents, true))
	if finalState.MarkdownReport == "" {
		log.Error("  警告: MarkdownReport 为空！")
	}
	replyMsg.Payload = report

	//messageBuffer.WriteString(finalState.MarkdownReport)

	return nil
}

func saveReport(ctx context.Context, title, report string) core.Attachment {
	attachment := core.Attachment{}
	attachment.ID = util.GetUUID()
	attachment.Name = title
	attachment.Size = len(report)
	attachment.MimeType = "text/markdown; charset=UTF-8"
	attachment.Icon = "book-open"
	attachment.URL = fmt.Sprintf("/attachment/%v", attachment.ID)
	attachment.Text = report
	ctx1 := orm.NewContextWithParent(ctx)
	err := orm.Save(ctx1, &attachment)
	if err != nil {
		panic(errors.Errorf("failed to save report: %v", err))
	}

	err = kv.AddValue(core.AttachmentKVBucket, []byte(attachment.ID), []byte(report))
	if err != nil {
		panic(err)
	}
	return attachment
}
