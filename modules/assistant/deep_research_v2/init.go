package deep_research

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

func RunDeepResearchV2(ctx context.Context, query string, config *core.DeepResearchConfig, reqMsg, replyMsg *core.ChatMessage, attachments []*core.Attachment, sender core.MessageSender) error {

	//response
	reasoningBuffer := strings.Builder{}
	messageBuffer := strings.Builder{}

	// completedState will be set after graph.Invoke so the defer can access collected chunks.
	var completedState *State

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
		// Persist all collected deep research chunks so the frontend can replay them from history.
		if completedState != nil && len(completedState.Chunks) > 0 {
			replyMsg.Details = append(replyMsg.Details, core.ProcessingDetails{
				Order:   10,
				Type:    common.DeepResearch,
				Payload: completedState.Chunks,
			})
		}
	}()

	log.Infof("Starting Deep-Research research agent, query: %s\n", query)

	graph, err := NewGraph()
	if err != nil {
		panic(errors.Errorf("Failed to create graph: %v", err))
	}

	initialState := &State{
		Config:      config,
		Sender:      sender,
		Attachments: attachments,
		Request: Request{
			Query: query,
		},
	}

	// Apply timeout if configured
	invokeCtx := ctx
	if config.Timeout != "" {
		if d, err := time.ParseDuration(config.Timeout); err == nil {
			var cancel context.CancelFunc
			invokeCtx, cancel = context.WithTimeout(ctx, d)
			defer cancel()
		}
	}

	// Point completedState to initialState now so the defer can persist
	// whatever chunks have been collected even if graph.Invoke panics
	// (e.g. on context cancellation).
	completedState = initialState

	result, err := graph.Invoke(invokeCtx, initialState)
	if err != nil {
		panic(errors.Errorf("Graph execution failed: %v", err))
	}

	finalState := result.(*State)
	completedState = finalState
	log.Info("\n=== Final Report ===")
	log.Info(finalState.MarkdownReport)

	reportFormat := config.ReportFormat
	if reportFormat == "" {
		reportFormat = "markdown"
	}

	var reportContent string
	var reportBytes []byte

	switch reportFormat {
	case "html":
		reportContent = finalState.HTMLReport
	case "pdf":
		reportBytes = finalState.PDFReport
	default:
		reportContent = finalState.MarkdownReport
	}

	attachment := saveReport(ctx, reportFormat, reportContent, reportBytes)

	report := util.MapStr{}
	report["title"] = attachment.Name
	report["url"] = attachment.URL
	report["created"] = attachment.Created
	report["attachment"] = attachment.ID
	report["format"] = reportFormat
	finalState.sendAndCollect(common.ResearchReporterEnd, util.MustToJSON(report))

	log.Info("Report generation completed:")
	switch reportFormat {
	case "html":
		log.Info("  HTMLReport length: ", len(finalState.HTMLReport))
	case "pdf":
		log.Info("  PDFReport length: ", len(finalState.PDFReport))
	default:
		log.Info("  MarkdownReport length: ", len(finalState.MarkdownReport))
	}
	log.Info("  ChapterOutline count: ", len(finalState.ChapterOutline))
	log.Info("  ChapterOutline: ", util.ToJson(finalState.ChapterOutline, true))
	log.Info("  ChapterContents count: ", len(finalState.ChapterContents))
	log.Info("  ChapterContents: ", util.ToJson(finalState.ChapterContents, true))
	if finalState.MarkdownReport == "" {
		log.Error("  WARNING: MarkdownReport is empty!")
	}
	replyMsg.Payload = report

	//messageBuffer.WriteString(finalState.MarkdownReport)

	return nil
}

// saveReport persists a generated research report as an attachment.
//
// report is the text content (markdown or HTML) and reportBytes holds the
// PDF binary. Exactly one is non-nil depending on reportFormat:
//   - markdown/html → report is set, reportBytes is nil
//   - pdf          → reportBytes is set, report is empty
func saveReport(ctx context.Context, reportFormat, report string, reportBytes []byte) core.Attachment {
	attachment := core.Attachment{}
	attachment.ID = util.GetUUID()

	var content []byte

	switch reportFormat {
	case "html":
		attachment.Name = "Research-Report.html"
		attachment.MimeType = "text/html; charset=UTF-8"
		attachment.Text = report
		attachment.Size = int64(len(report))
		content = []byte(report)
	case "pdf":
		attachment.Name = "Research-Report.pdf"
		attachment.MimeType = "application/pdf"
		attachment.Size = int64(len(reportBytes))
		content = reportBytes
	default:
		attachment.Name = "Research-Report.md"
		attachment.MimeType = "text/markdown; charset=UTF-8"
		attachment.Text = report
		attachment.Size = int64(len(report))
		content = []byte(report)
	}
	attachment.Icon = "book-open"
	attachment.URL = fmt.Sprintf("/attachment/%v", attachment.ID)
	ctx1 := orm.NewContextWithParent(ctx)
	err := orm.Save(ctx1, &attachment)
	if err != nil {
		panic(errors.Errorf("failed to save report: %v", err))
	}

	err = kv.AddValue(core.AttachmentKVBucket, []byte(attachment.ID), content)
	if err != nil {
		panic(err)
	}
	return attachment
}
