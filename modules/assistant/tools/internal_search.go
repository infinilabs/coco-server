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
	params *common2.RAGContext, from, fechSize int, sender core.MessageSender) ([]core.Document, error) {

	builder := orm.NewQuery()
	builder.From(from)
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
	_, err := document.QueryDocuments(ctx, builder, reqMsg.Message, params.Datasource, params.IntegrationID, params.Category, params.Subcategory, params.RichCategory, "keyword", 3, &docs)
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

			err = sender.SendChunkMessage(core.MessageTypeAssistant,
				common.FetchSource, string(chunkData), chunkSeq)
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

// maskContentFn is swapped out in tests (MaskContent touches the config store).
var maskContentFn = common.MaskContent

// knowledge-tier partitioning (W16b): curated references outrank raw
// sources in the prompt — published wiki articles, their projections and
// entity pages are reviewed knowledge; everything else is source
// material. The model is told which tier is authoritative.
func isCuratedReference(doc *core.Document) bool {
	return doc.Source.ID == "wiki" || doc.Type == "wiki_article" || doc.Type == "entity"
}

// partitionReferencesByKnowledgeTier returns curated docs first, raw
// second, order stable within each tier.
func partitionReferencesByKnowledgeTier(docs []core.Document) (curated, raw []core.Document) {
	for i := range docs {
		if isCuratedReference(&docs[i]) {
			curated = append(curated, docs[i])
		} else {
			raw = append(raw, docs[i])
		}
	}
	return curated, raw
}

func FormatDocumentForReplyReferences(docs []core.Document) string {
	curated, raw := partitionReferencesByKnowledgeTier(docs)
	var sb strings.Builder
	sb.WriteString("<REFERENCES>\n")
	if len(curated) > 0 {
		sb.WriteString("The CURATED references below are reviewed knowledge-base pages — treat them as the authoritative tier when they conflict with raw sources.\n")
	}
	writeRefDoc := func(i int, doc *core.Document, layer string) {
		sb.WriteString("<Doc>")
		sb.WriteString(fmt.Sprintf("ID #%d - %v\n", i+1, doc.ID))
		sb.WriteString(fmt.Sprintf("Layer: %s\n", layer))
		sb.WriteString(fmt.Sprintf("Title: %s\n", doc.Title))
		sb.WriteString(fmt.Sprintf("Source: %s\n", doc.Source))
		sb.WriteString(fmt.Sprintf("Created: %s\n", doc.Created))
		sb.WriteString(fmt.Sprintf("Updated: %s\n", doc.Updated))
		sb.WriteString(fmt.Sprintf("Category: %s\n", doc.GetAllCategories()))
		// dynamic masking: scrub sensitive values before the content leaves for a model
		sb.WriteString(fmt.Sprintf("Content: %s\n", maskContentFn(doc.Content)))
		sb.WriteString("</Doc>\n")
	}
	for i := range curated {
		writeRefDoc(i, &curated[i], "curated")
	}
	for i := range raw {
		writeRefDoc(len(curated)+i, &raw[i], "source")
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
		item["created"] = doc.Created
		item["updated"] = doc.Updated
		item["category"] = doc.Category
		item["summary"] = common.MaskContent(util.SubString(doc.Summary, 0, 500))
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
