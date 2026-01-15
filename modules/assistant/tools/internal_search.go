package tools

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	common2 "infini.sh/coco/modules/assistant/common"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/document"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

func InitialDocumentBriefSearch(ctx context.Context, userID string, reqMsg, replyMsg *core.ChatMessage,
	params *common2.RAGContext, fechSize int, sender core.MessageSender) ([]core.Document, error) {

	builder := orm.NewQuery()
	builder.Size(fechSize)

	//merge the user defined query to filter
	if params.AssistantCfg.Datasource.Enabled && params.AssistantCfg.Datasource.Filter != nil {
		log.Debug("custom filter:", params.AssistantCfg.Datasource.Filter)
		q := util.MapStr{}
		q["query"] = params.AssistantCfg.Datasource.Filter
		builder.SetRequestBodyBytes(util.MustToJSONBytes(q))
		builder.EnableBodyBytes()
	}

	if params.QueryIntent != nil && len(params.QueryIntent.Query) > 0 {
		builder.Should(orm.TermsQuery("combined_fulltext", params.QueryIntent.Keyword))
		builder.Should(orm.TermsQuery("combined_fulltext", params.QueryIntent.Query))
	}

	teamsID := GetTeamsIDByUserID(ctx, userID)
	if len(teamsID) > 0 {
		ctx = context.WithValue(ctx, orm.TeamsIDKey, teamsID)
	}
	ctx = context.WithValue(ctx, orm.OwnerIDKey, userID)

	docs := []core.Document{}
	_, err := document.QueryDocuments(ctx, builder, reqMsg.Message, params.Datasource, params.IntegrationID, params.Category, params.Subcategory, params.RichCategory, &docs)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	{
		simplifiedReferences := formatDocumentReferencesToDisplay(docs)
		const chunkSize = 512
		totalLen := len(simplifiedReferences)

		for chunkSeq := 0; chunkSeq*chunkSize < totalLen; chunkSeq++ {
			start := chunkSeq * chunkSize
			end := start + chunkSize
			if end > totalLen {
				end = totalLen
			}

			chunkData := simplifiedReferences[start:end]

			chunkMsg := core.NewMessageChunk(params.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID,
				common.FetchSource, string(chunkData), chunkSeq)

			err = sender.SendMessage(chunkMsg)
			if err != nil {
				log.Error(err)
				return nil, err
			}
		}
	}

	fetchedDocs := formatDocumentForPick(docs)
	{
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("<Payload total=%v>\n", len(docs)))
		sb.WriteString(util.MustToJSON(fetchedDocs))
		sb.WriteString("</Payload>")
		params.SourceDocsSummaryBlock = sb.String()
	}
	replyMsg.Details = append(replyMsg.Details, core.ProcessingDetails{Order: 20, Type: common.FetchSource, Payload: fetchedDocs})
	return docs, err
}

func GetTeamsIDByUserID(ctx context.Context, userID string) []string {
	if global.Env().SystemConfig.WebAppConfig.Security.Managed {

		sessionUser := security.MustGetUserFromContext(ctx)

		profileKey := fmt.Sprintf("%v:%v", sessionUser.MustGetString(orm.TenantIDKey), userID)

		//get profile
		data, err := kv.GetValue(core.UserProfileBucketKey, []byte(profileKey))
		if err != nil {
			panic(err)
		}

		p := &security.UserProfile{}
		util.MustFromJSONBytes(data, p)
		v, ok := p.GetSystemValue(orm.TeamsIDKey)
		if ok {
			v, ok := v.([]interface{})
			if ok {
				out := []string{}
				for _, v1 := range v {
					x, ok := v1.(string)
					if ok {
						out = append(out, x)
					}
				}
				return out
			}
		}
	}
	return []string{}
}

func formatDocumentForReplyReferences(docs []core.Document) string {
	var sb strings.Builder
	sb.WriteString("<REFERENCES>\n")
	for i, doc := range docs {
		sb.WriteString(fmt.Sprintf("<Doc>"))
		sb.WriteString(fmt.Sprintf("ID #%d - %v\n", i+1, doc.ID))
		sb.WriteString(fmt.Sprintf("Title: %s\n", doc.Title))
		sb.WriteString(fmt.Sprintf("Source: %s\n", doc.Source))
		sb.WriteString(fmt.Sprintf("Updated: %s\n", doc.Updated))
		sb.WriteString(fmt.Sprintf("Category: %s\n", doc.GetAllCategories()))
		sb.WriteString(fmt.Sprintf("Content: %s\n", doc.Content))
		sb.WriteString(fmt.Sprintf("</Doc>\n"))

	}
	sb.WriteString("</REFERENCES>")
	return sb.String()
}

func formatDocumentReferencesToDisplay(docs []core.Document) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<Payload total=%v>\n", len(docs)))
	outDocs := []util.MapStr{}
	for _, doc := range docs {
		item := util.MapStr{}
		item["id"] = doc.ID
		item["title"] = doc.Title
		item["source"] = doc.Source
		item["icon"] = doc.Icon
		item["url"] = doc.URL
		outDocs = append(outDocs, item)
	}
	sb.WriteString(util.MustToJSON(outDocs))
	sb.WriteString("</Payload>")
	return sb.String()
}

func formatDocumentForPick(docs []core.Document) []util.MapStr {
	outDocs := []util.MapStr{}
	for _, doc := range docs {
		item := util.MapStr{}
		item["id"] = doc.ID
		item["title"] = doc.Title
		item["updated"] = doc.Updated
		item["category"] = doc.Category
		item["summary"] = util.SubString(doc.Summary, 0, 500)
		item["url"] = doc.URL
		outDocs = append(outDocs, item)
	}
	return outDocs
}

func fetchDocuments(query *orm.Query) ([]core.Document, error) {
	var docs []core.Document
	err, _ := orm.SearchWithJSONMapper(&docs, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documents: %w", err)
	}
	return docs, nil
}
