package tools

import (
	"context"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/core"
	common2 "infini.sh/coco/modules/assistant/common"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/util"
)

func PickingDocuments(ctx context.Context, reqMsg, replyMsg *core.ChatMessage,
	params *common2.RAGContext, docs []core.Document, sender core.MessageSender) ([]core.Document, error) {

	if len(docs) == 0 {
		return nil, nil
	}

	err := sender.SendChunkMessage(core.MessageTypeAssistant, common.PickSource, string(""), 0)
	if err != nil {
		panic(err)
	}

	promptTemplate := common.PickingDocPromptTemplate
	if params.AssistantCfg.DeepThinkConfig.PickingDocModel.PromptConfig.PromptTemplate != "" {
		promptTemplate = params.AssistantCfg.DeepThinkConfig.PickingDocModel.PromptConfig.PromptTemplate
	}
	// Create the prompt template
	inputValues := map[string]any{
		"query":  reqMsg.Message,
		"intent": util.MustToJSON(params.QueryIntent),
		"docs":   params.SourceDocsSummaryBlock,
	}
	finalPrompt, err := langchain.GetPromptStringByTemplateArgs(&params.AssistantCfg.DeepThinkConfig.PickingDocModel, promptTemplate, []string{"query", "intent", "summary"}, inputValues)
	if err != nil {
		panic(err)
	}
	content := []llms.MessageContent{
		llms.TextParts(
			llms.ChatMessageTypeSystem,
			finalPrompt,
		),
	}

	log.Debug("start filtering documents")
	var pickedDocsBuffer = strings.Builder{}
	var chunkSeq = 0
	//llm := langchain.GetLLM(params.pickingDocProvider.BaseURL, params.pickingDocProvider.APIType, params.pickingDocModel.Name, params.pickingDocProvider.APIKey, params.AssistantCfg.Keepalive)

	llm, err := langchain.SimplyGetLLM(params.AssistantCfg.DeepThinkConfig.PickingDocModel.ProviderID, params.AssistantCfg.DeepThinkConfig.PickingDocModel.Name, "")
	if err != nil {
		panic(err)
	}

	//options:=langchain.GetLLOptions(&params.AssistantCfg.DeepThinkConfig.PickingDocModel)

	log.Trace(content)
	if _, err := llm.GenerateContent(ctx, content,
		llms.WithMaxLength(util.GetIntOrDefault(params.AssistantCfg.DeepThinkConfig.PickingDocModel.Settings.MaxLength, 32768)),
		llms.WithMaxTokens(util.GetIntOrDefault(params.AssistantCfg.DeepThinkConfig.PickingDocModel.Settings.MaxTokens, 32768)),
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			if len(chunk) > 0 {
				chunkSeq++
				pickedDocsBuffer.Write(chunk)
				err = sender.SendChunkMessage(core.MessageTypeAssistant, common.PickSource, string(chunk), chunkSeq)
				if err != nil {
					return err
				}
			}
			return nil
		})); err != nil {
		return nil, err
	}

	log.Debug(pickedDocsBuffer.String())

	pickeDocs, err := langchain.PickedDocumentFromString(pickedDocsBuffer.String())
	if err != nil {
		return nil, err
	}

	log.Debug("filter document results:", pickeDocs)

	docsMap := map[string]core.Document{}
	for _, v := range docs {
		docsMap[v.ID] = v
	}

	var pickedDocIDS []string
	var pickedFullDoc = []core.Document{}
	var validPickedDocs = []langchain.PickedDocument{}
	for _, v := range pickeDocs {
		x, v1 := docsMap[v.ID]
		if v1 {
			pickedDocIDS = append(pickedDocIDS, v.ID)
			pickedFullDoc = append(pickedFullDoc, x)
			validPickedDocs = append(validPickedDocs, v)
			log.Debug("pick doc:", x.ID, ",", x.Title)
		} else {
			log.Error("wrong doc id, doc is missing")
		}
	}

	{
		detail := core.ProcessingDetails{Order: 30, Type: common.PickSource, Payload: validPickedDocs}
		replyMsg.Details = append(replyMsg.Details, detail)
	}

	params.PickedDocIDS = pickedDocIDS

	log.Debug("valid picked document results:", validPickedDocs)

	//replace to picked one
	docs = pickedFullDoc
	return docs, err
}
