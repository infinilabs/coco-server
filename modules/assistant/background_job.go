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

package assistant

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"infini.sh/coco/modules/assistant/rag"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/search"
	"infini.sh/framework/core/api/websocket"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"net/http"
	"strings"
)

func (h APIHandler) handleMessage(req *http.Request, sessionID, message string) (*ChatMessage, error) {

	var (
		from         = h.GetIntOrDefault(req, "from", 0)
		size         = h.GetIntOrDefault(req, "size", 10)
		datasource   = h.GetParameterOrDefault(req, "datasource", "")
		category     = h.GetParameterOrDefault(req, "category", "")
		username     = h.GetParameterOrDefault(req, "username", "")
		userid       = h.GetParameterOrDefault(req, "userid", "")
		tags         = h.GetParameterOrDefault(req, "tags", "")
		subcategory  = h.GetParameterOrDefault(req, "subcategory", "")
		richCategory = h.GetParameterOrDefault(req, "rich_category", "")
		field        = h.GetParameterOrDefault(req, "search_field", "title")
		source       = h.GetParameterOrDefault(req, "source_fields", "*")
	)

	searchDB := h.GetBoolOrDefault(req, "search", false)
	//deepThink := h.GetBoolOrDefault(req, "deep_thinking", false)

	log.Debug("handle message:", message)

	obj := ChatMessage{
		SessionID:   sessionID,
		MessageType: MessageTypeUser,
		Message:     message,
	}

	//TODO
	if searchDB {
		obj.Parameters = util.MapStr{}
		obj.Parameters["search"] = searchDB
	}

	//save user's message
	err := orm.Create(nil, &obj)
	if err != nil {
		return nil, err
	}

	//send to background job
	webSocketID, err := h.GetUserWebsocketID(req)
	if err == nil && webSocketID != "" {

		//de-duplicate background task per-session, cancelable
		taskID := task.RunWithinGroup("assistant-session", func(taskCtx context.Context) error {
			//timeout for 30 seconds

			log.Debugf("place a assistant background job for session: %v, websocket: %v ", sessionID, webSocketID)

			//1. expand and rewrite the query
			// use the title and summary to judge which document need to fetch in-depth, also the updated time to check the data is fresh or not
			// pick N related documents and combine with the memory and the near chat history as the chat context
			//2. summary previous history chat as context, update as memory
			//3. assemble with the agent's role setting
			//4. send to LLM

			//ollamaConfig := common.AppConfig().OllamaConfig
			//llm, err := ollama.New(
			//	ollama.WithServerURL(ollamaConfig.Endpoint),
			//	ollama.WithModel(ollamaConfig.Model),
			//	ollama.WithKeepAlive(ollamaConfig.Keepalive))

			//TODO, more options exposed to config
			if err != nil {
				log.Error(err)
				return err
			}
			ctx := context.Background()

			chunkSeq := 0
			messageID := util.GetUUID()
			requestMessageID := obj.ID

			details := []ProcessingDetails{}

			//query intent
			var queryIntentStr string

			//queryIntent := fmt.Sprintf("- 关键字: \n %v\n- 问题类型: %v\n- 用户意图: %v", message, "常规问题", "用户就是想随便问问")
			var queryIntent *rag.QueryAnalysis
			{
				log.Info("开始进行意图识别")
				queryIntentBuffer := strings.Builder{}
				//chunkSeq++
				//msg := NewMessageChunk(sessionID, messageID, MessageTypeAssistant, requestMessageID, QueryIntent, string("开始进行意图识别"), chunkSeq)
				//websocket.SendPrivateMessage(webSocketID, util.MustToJSON(msg))
				//
				content := []llms.MessageContent{
					llms.TextParts(llms.ChatMessageTypeSystem, "You are an AI assistant trained to understand and analyze user queries. The user has provided the following query:"),
					llms.TextParts(llms.ChatMessageTypeHuman, message),
					llms.TextParts(llms.ChatMessageTypeSystem, "Please analyze the query and identify the user's primary intent. "+
						"Determine if they are looking for information, making a request, or seeking clarification. "+
						"Category the intent in </Category>, brief the </Intent>, and rephrase the query in several different forms to improve clarity. "+
						"Provide possible variations of the query in <Query/> and identify relevant keywords in </Keyword> in JSON array format. "+
						"Provide possible related of the query in <Suggestion/> and expand the related query for query suggestion. "+
						"Please make sure the output is concise, well-organized, and easy to process."+
						"Please present these possible query and keyword items in both English and Chinese."+
						"if the possible query is in English, keep the original English one, and translate it to Chinese and keep it as a new query, to be clear, you should output: [Apple, 苹果], neither just `Apple` nor just `苹果`."+
						"Wrap the JSON result in <JSON></JSON> tags. "+
						"Your output should look like this format:\n"+
						"<JSON>"+
						"{\n"+
						"  \"category\": \"<Intent's Category>\",\n"+
						"  \"intent\": \"<User's Intent>\",\n"+
						"  \"query\": [\n"+
						"    \"<新的查询 1>\",\n"+
						"    \"<Rephrased Query 2>\",\n"+
						"    \"<Rephrased Query 3>\"\n"+
						"  ],\n"+
						"  \"keyword\": [\n"+
						"    \"<关键字 1>\",\n"+
						"    \"<Keyword 2>\",\n"+
						"    \"<Keyword 3>\"\n"+
						"  ],\n"+
						"  \"suggestion\": [\n"+
						"    \"<Suggest Query 1>\",\n"+
						"    \"<Suggest Query 2>\",\n"+
						"    \"<Suggest Query 3>\"\n"+
						"  ]\n"+
						"}"+
						"</JSON>"),
				}

				llm := getLLM("tongyi-intent-detect-v3")
				if _, err := llm.GenerateContent(ctx, content,
					llms.WithMaxTokens(1024),
					llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
						if len(chunk) > 0 {
							chunkSeq++
							queryIntentBuffer.Write(chunk)
							msg := NewMessageChunk(sessionID, messageID, MessageTypeAssistant, requestMessageID, QueryIntent, string(chunk), chunkSeq)
							err := websocket.SendPrivateMessage(webSocketID, util.MustToJSON(msg))
							if err != nil {
								panic(err)
							}
						}
						return nil
					})); err != nil {
					log.Error(err)
				}
				log.Info("结束意图识别")
				if queryIntentBuffer.Len() > 0 {
					//extract the category and query
					queryIntentStr = queryIntentBuffer.String()
					//log.Info("意图识别结果:\n", queryIntentStr)
					queryIntent, err = rag.QueryAnalysisFromString(queryIntentStr)
					if err != nil {
						panic(err)
					}
					detail := ProcessingDetails{Order: 10, Type: QueryIntent, Payload: queryIntent}
					details = append(details, detail)

				}
			}

			var query *orm.Query
			if searchDB {
				mustClauses := search.BuildMustClauses(datasource, category, subcategory, richCategory, username, userid)
				//should_clauses := search.BuildMustClauses(datasource, category, subcategory, richCategory, username, userid)
				//
				var shouldClauses interface{}
				if queryIntent != nil && len(queryIntent.Query) > 0 {
					//log.Info("queryIntent:", queryIntent.Query)
					shouldClauses = search.BuildShouldClauses(queryIntent.Query, queryIntent.Keyword)
				}

				//initial fetch size
				size = 50
				query = search.BuildTemplatedQuery(from, size, mustClauses, shouldClauses, field, obj.Message, source, tags)
			}

			var references string
			var pickedDocIDS []string
			var historyStr = strings.Builder{}

			//Retrieve related documents from background server
			if searchDB && query != nil {
				docs, err := fetchDocuments(query)
				log.Infof("命中 %v 个文档", len(docs))
				if err != nil {
					log.Errorf("Failed to fetch documents from DB: %v", err)
					// Proceed without RAG
				} else if len(docs) > 0 {

					//references = formatDocumentReferences(docs)
					simpliedReferences := formatDocumentReferencesToDisplay(docs)
					//TODO save messageBuffer.WriteString(simpliedReferences)

					chunkSeq++
					msg := NewMessageChunk(sessionID, messageID, MessageTypeAssistant, requestMessageID, FetchSource, string(simpliedReferences), chunkSeq)
					err := websocket.SendPrivateMessage(webSocketID, util.MustToJSON(msg))
					if err != nil {
						panic(err)
					}

					{
						fetchedDocs := formatDocumentForPick(docs)
						var simpleDocsStr string
						{
							var sb strings.Builder
							sb.WriteString(fmt.Sprintf("<Payload total=%v>\n", len(docs)))
							sb.WriteString(util.MustToJSON(fetchedDocs))
							sb.WriteString("</Payload>")
							simpleDocsStr = sb.String()
						}

						detail := ProcessingDetails{Order: 20, Type: FetchSource, Payload: fetchedDocs}
						details = append(details, detail)

						//get chat history
						history, err := getChatHistoryBySessionInternal(sessionID)
						log.Error("history:", history, err)
						historyStr.WriteString("<conversation>")
						//<summary>
						//session history summary within 500 words TODO
						//</summary>
						//<recent>
						//recent 10 Q&A history records //TODO configurable
						//</recent>
						for _, v := range history {
							historyStr.WriteString(v.MessageType + ": " + util.SubStringWithSuffix(v.Message, 500, "...")) //TODO 问题是否准确来判断是否采用
							if v.DownVote > 0 {
								historyStr.WriteString(fmt.Sprintf("(%v people up voted this answer)", v.UpVote))
							}
							if v.DownVote > 0 {
								historyStr.WriteString(fmt.Sprintf("(%v people down voted this answer)", v.DownVote))
							}
							historyStr.WriteString("\n")
						}
						historyStr.WriteString("</conversation>")

						content := []llms.MessageContent{
							llms.TextParts(llms.ChatMessageTypeSystem, "You are an AI assistant trained to understand and analyze user queries. "),

							//get history
							llms.TextParts(llms.ChatMessageTypeSystem, "You will be given a conversation below and a follow up question. "+
								"You need to rephrase the follow-up question if needed so it is a standalone question that can be used by the LLM to search the knowledge base for information.\n"+
								"Conversation: "),
							llms.TextParts(llms.ChatMessageTypeSystem, historyStr.String()),
							//end history

							llms.TextParts(llms.ChatMessageTypeSystem, "The user has provided the following query:"),
							llms.TextParts(llms.ChatMessageTypeHuman, message),

							llms.TextParts(llms.ChatMessageTypeSystem, "The primary intent behind this query is:"),
							llms.TextParts(llms.ChatMessageTypeSystem, string(queryIntentStr)),

							llms.TextParts(llms.ChatMessageTypeSystem, "The following documents might be related to answering the user's query:"),

							llms.TextParts(llms.ChatMessageTypeSystem, string(simpleDocsStr)),

							llms.TextParts(llms.ChatMessageTypeSystem, "\nPlease review these documents and identify which ones best match the user's query. "+
								"Choose no more than 5 relevant documents. These documents may be entirely unrelated, so prioritize those that provide direct answers or valuable context."+
								"If the document is unrelated not certain, don't include it."+
								" For each document, provide a brief explanation of why it was selected, categorizing it in </Document> tags."+
								" Make sure the output is concise and easy to process."+
								" Wrap the JSON result in <JSON></JSON> tags."+
								" The expected output format is:\n"+
								"<JSON>\n"+
								"[\n"+
								" { \"id\": \"<id of Doc 1>\", \"title\": \"<title of Doc 1>\", \"explain\": \"<Explain for Doc 1>\"  },\n"+
								" { \"id\": \"<id of Doc 2>\", \"title\": \"<title of Doc 2>\", \"explain\": \"<Explain for Doc 2>\"  },\n"+
								"]"+
								"</JSON>"),
						}

						log.Info("开始筛选文档")
						var pickedDocsBuffer = strings.Builder{}
						llm := getLLM("deepseek-r1-distill-qwen-32b")
						if _, err := llm.GenerateContent(ctx, content,
							llms.WithMaxTokens(32768),
							llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
								if len(chunk) > 0 {
									chunkSeq++
									pickedDocsBuffer.Write(chunk)
									msg := NewMessageChunk(sessionID, messageID, MessageTypeAssistant, requestMessageID, PickSource, string(chunk), chunkSeq)
									err := websocket.SendPrivateMessage(webSocketID, util.MustToJSON(msg))
									if err != nil {
										panic(err)
									}
								}
								return nil
							})); err != nil {
							log.Error(err)
						}

						pickeDocs, err := rag.PickedDocumentFromString(pickedDocsBuffer.String())
						if err != nil {
							panic(err)
						}

						log.Info("筛选文档结果:", pickedDocsBuffer.String())
						{
							detail := ProcessingDetails{Order: 30, Type: PickSource, Payload: pickeDocs}
							details = append(details, detail)
						}

						docsMap := map[string]common.Document{}
						for _, v := range docs {
							docsMap[v.ID] = v
						}

						var pickedFullDoc = []common.Document{}
						for _, v := range pickeDocs {
							x, v1 := docsMap[v.ID]
							if v1 {
								pickedDocIDS = append(pickedDocIDS, v.ID)
								pickedFullDoc = append(pickedFullDoc, x)
								//log.Info("pick doc:", x.ID,",",x.Title)
							} else {
								log.Error("wrong doc id, doc is missing")
							}
						}

						//replace to picked one
						docs = pickedFullDoc
					}

					{
						if len(pickedDocIDS) > 0 {
							var query = orm.Query{}
							query.Conds = orm.And(orm.InStringArray("_id", pickedDocIDS))

							pickedFullDoc, err := fetchDocuments(&query)

							strBuilder := strings.Builder{}
							for _, v := range pickedFullDoc {
								str := "Obtaining and analyzing documents in depth:  " + string(v.Title) + "\n"
								strBuilder.WriteString(str)
								msg = NewMessageChunk(sessionID, messageID, MessageTypeAssistant, requestMessageID, DeepRead, str, chunkSeq)
								err = websocket.SendPrivateMessage(webSocketID, util.MustToJSON(msg))
								if err != nil {
									panic(err)
								}
							}

							detail := ProcessingDetails{Order: 40, Type: DeepRead, Description: strBuilder.String()}
							details = append(details, detail)

							references = formatDocumentForReplyReferences(pickedFullDoc)
						}
					}
				}
				log.Infof("Fetched %v docs with query: %v", len(docs), query)
			}

			prompt := fmt.Sprintf(`You are a friendly assistant designed to help users access and understand their personal or company data. 
Your responses should be clear, concise, and based solely on the information provided below. 
If the information is insufficient, please indicate that you need more details to assist effectively.

Conversation: %s

Query: %s

Data:
%s`, historyStr.String(), message, references)

			// Prepare the system message
			content := []llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeSystem, "You are a personal AI assistant designed by Coco AI(https://coco.rs), the backend team is behind INFINI Labs(https://infinilabs.com)."),
			}

			// Append the user's message
			content = append(content, llms.TextParts(llms.ChatMessageTypeHuman, prompt))

			//log.Debug(content)

			//response
			reasoningBuffer := strings.Builder{}
			messageBuffer := strings.Builder{}

			llm := getLLM("deepseek-r1") //deepseek-r1 /deepseek-v3
			appConfig := common.AppConfig()
			completion, err := llm.GenerateContent(ctx, content,
				llms.WithMaxTokens(appConfig.LLMConfig.Parameters.MaxTokens),
				llms.WithMaxLength(appConfig.LLMConfig.Parameters.MaxLength),

				llms.WithStreamingReasoningFunc(func(ctx context.Context, reasoningChunk []byte, chunk []byte) error {
					// Use taskCtx here to check for cancellation or other context-specific logic
					select {
					case <-ctx.Done(): // Check if the task has been canceled or has expired
						log.Warnf("Task for message %v canceled", messageID)
						return taskCtx.Err() // Return the context error (canceled or deadline exceeded)
					case <-taskCtx.Done(): // Check if the task has been canceled or has expired
						log.Warnf("Task for message %v canceled", messageID)
						return taskCtx.Err() // Return the context error (canceled or deadline exceeded)
					default:

						//Handle the <Think> part
						if len(reasoningChunk) > 0 {
							chunkSeq += 1
							reasoningBuffer.Write(reasoningChunk)
							msg := NewMessageChunk(sessionID, messageID, MessageTypeAssistant, requestMessageID, Think, string(reasoningChunk), chunkSeq)
							//log.Info(util.MustToJSON(msg))
							err = websocket.SendPrivateMessage(webSocketID, util.MustToJSON(msg))
							if err != nil {
								panic(err)
							}
							//TODO
							//save buffer
							return nil
						}

						//Handle response
						if len(chunk) > 0 {
							chunkSeq += 1

							msg := NewMessageChunk(sessionID, messageID, MessageTypeAssistant, requestMessageID, Response, string(chunk), chunkSeq)
							err = websocket.SendPrivateMessage(webSocketID, util.MustToJSON(msg))
							if err != nil {
								panic(err)
							}

							//log.Debug(msg)
							messageBuffer.Write(chunk)
						}

						return nil
					}

				}),

				////this part works for ollama
				//llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				//
				//	var txtMsg string
				//	if !sentSource {
				//		if simpliedReferences != "" {
				//			txtMsg = simpliedReferences
				//		}
				//		sentSource = true
				//	}
				//	txtMsg += string(chunk)
				//
				//	chunkSeq += 1
				//	msg := util.MustToJSON(util.MapStr{
				//		"session_id":       sessionID,
				//		"message_id":       messageID,
				//		"message_type":     MessageTypeAssistant,
				//		"reply_to_message": requestMessageID,
				//		"chunk_sequence":   chunkSeq,
				//		"message_chunk":    txtMsg,
				//	})
				//	messageBuffer.Write(chunk)
				//	websocket.SendPrivateMessage(webSocketID, msg)
				//	return nil
				//}),

				llms.WithTemperature(0.8),
			)
			if err != nil {
				log.Error(err)
				return err
			}
			_ = completion

			chunkSeq += 1

			{
				detail := ProcessingDetails{Order: 50, Type: Think, Description: reasoningBuffer.String()}
				details = append(details, detail)
			}

			{
				detail := ProcessingDetails{Order: 60, Type: Response, Description: messageBuffer.String()}
				details = append(details, detail)
			}

			msg := NewMessageChunk(sessionID, messageID, MessageTypeSystem, requestMessageID, ReplyEnd, "assistant finished output", chunkSeq)
			err = websocket.SendPrivateMessage(webSocketID, util.MustToJSON(msg))
			if err != nil {
				panic(err)
			}

			log.Info(util.MustToJSON(details))

			//save response message to system
			if len(details) > 0 || messageBuffer.Len() > 0 {
				obj = ChatMessage{
					SessionID:   sessionID,
					MessageType: MessageTypeAssistant,
					Message:     messageBuffer.String(),
					Details:     details,
				}
				obj.ID = messageID

				err = orm.Save(nil, &obj)
				if err != nil {
					log.Error(err)
					return err
				}
			} else {
				log.Warnf("seems empty reply for query:", message)
			}

			log.Info("async reply task done for query:", message)
			return nil
		})

		log.Infof("save taskid: %v, sessionID:%v", taskID, sessionID)

		inflightMessages.Store(sessionID, MessageTask{TaskID: taskID, WebsocketID: webSocketID})
	} else {
		log.Warnf("no websocket: [%v] found for session: %v ", webSocketID, sessionID)
		//TODO save to buffer, provide API to pulling data
	}

	return &obj, nil
}

func formatDocumentForReplyReferences(docs []common.Document) string {
	var sb strings.Builder
	sb.WriteString("<REFERENCES>\n")
	for i, doc := range docs {
		sb.WriteString(fmt.Sprintf("<Doc>"))
		sb.WriteString(fmt.Sprintf("ID #%d - %v\n", i+1, doc.ID))
		sb.WriteString(fmt.Sprintf("Title: %s\n", doc.Title))
		sb.WriteString(fmt.Sprintf("Source: %s\n", doc.Source))
		sb.WriteString(fmt.Sprintf("Updated: %s\n", doc.Updated))
		sb.WriteString(fmt.Sprintf("Category: %s\n", doc.GetAllCategories()))
		//sb.WriteString(fmt.Sprintf("Summary: %s\n", doc.Summary))
		sb.WriteString(fmt.Sprintf("Content: %s\n", doc.Content))
		sb.WriteString(fmt.Sprintf("</Doc>\n"))

	}
	sb.WriteString("</REFERENCES>")
	return sb.String()
}

func formatDocumentReferencesToDisplay(docs []common.Document) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<Payload total=%v>\n", len(docs)))
	outDocs := []util.MapStr{}
	for _, doc := range docs {
		item := util.MapStr{}
		item["id"] = doc.ID
		item["title"] = doc.Title
		item["source"] = doc.Source
		item["updated"] = doc.Updated
		item["category"] = doc.Category
		item["summary"] = doc.Summary
		item["icon"] = doc.Icon
		item["size"] = doc.Size
		item["thumbnail"] = doc.Thumbnail
		item["url"] = doc.URL
		outDocs = append(outDocs, item)
	}
	sb.WriteString(util.MustToJSON(outDocs))
	sb.WriteString("</Payload>")
	return sb.String()
}

func formatDocumentForPick(docs []common.Document) []util.MapStr {
	outDocs := []util.MapStr{}
	for _, doc := range docs {
		item := util.MapStr{}
		item["id"] = doc.ID
		item["title"] = doc.Title
		item["updated"] = doc.Updated
		item["category"] = doc.Category
		item["summary"] = util.SubString(doc.Summary, 0, 500)
		outDocs = append(outDocs, item)
	}
	return outDocs
}

func fetchDocuments(query *orm.Query) ([]common.Document, error) {
	var docs []common.Document
	err, _ := orm.SearchWithJSONMapper(&docs, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documents: %w", err)
	}
	return docs, nil
}

func getLLM(model string) *openai.LLM {
	if model == "" {
		model = common.AppConfig().LLMConfig.DefaultModel
	}
	llm, err := openai.New(
		openai.WithToken(common.AppConfig().LLMConfig.Token),
		openai.WithBaseURL(common.AppConfig().LLMConfig.Endpoint),
		openai.WithModel(model),
	)
	if err != nil {
		panic(err)
	}
	return llm
}
