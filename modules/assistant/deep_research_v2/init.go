package deep_research

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/errors"
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
			log.Error("update message to: ", messageBuffer.String())
			replyMsg.Message = messageBuffer.String()
		} else {
			log.Warnf("seems empty reply for query: %v", replyMsg)
		}
		if reasoningBuffer.Len() > 0 {
			detail := core.ProcessingDetails{Order: 50, Type: common.Think, Description: reasoningBuffer.String()}
			replyMsg.Details = append(replyMsg.Details, detail)
		}
	}()

	fmt.Printf("正在启动 Deer-Flow 研究代理，查询内容：%s\n", query)

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
	fmt.Println("\n=== Final Report ===")
	fmt.Println(finalState.MarkdownReport)

	log.Info("报告生成完成：")
	log.Info("  MarkdownReport 长度: ", len(finalState.MarkdownReport))
	log.Info("  FinalReport 长度: ", len(finalState.FinalReport))
	log.Info("  章节大纲数量: ", len(finalState.ChapterOutline))
	log.Info("  ChapterContents 数量: ", len(finalState.ChapterContents))
	if finalState.MarkdownReport == "" {
		log.Error("  警告: MarkdownReport 为空！")
	}

	messageBuffer.WriteString(finalState.FinalReport)

	return nil
}
