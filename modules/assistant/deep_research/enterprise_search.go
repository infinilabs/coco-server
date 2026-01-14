package deep_research

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/document"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

type EnterpriseSearchTool struct {
}

// Name returns the tool name
func (t *EnterpriseSearchTool) Name() string {
	return "enterprise_search"
}

// Description returns the tool description
func (t *EnterpriseSearchTool) Description() string {
	return "Search the internal enterprise search engine for information. Input should be a search query string."
}

// Call executes the search
func (t *EnterpriseSearchTool) Call(ctx context.Context, input string) (string, error) {
	log.Info("start call EnterpriseSearchTool:", input)
	defer log.Info("end call EnterpriseSearchTool:", input)

	userInfo := security.MustGetUserFromContext(ctx)
	log.Error("hit enterprise_search_tool, MustGetUserID: ", userInfo.MustGetUserID())

	// Format results
	var results []string

	builder := orm.NewQuery()
	output := []core.Document{}
	_, err := document.InternalQueryDocuments(ctx, builder, input, "", "", &output)
	if err != nil {
		panic(err)
	}

	log.Error("hit enterprise_search_tool, output: ", util.ToJson(output, true))

	for i, result := range output {
		results = append(results, fmt.Sprintf(
			"[Result %d]\nTitle: %s\nURL: %s\nContent: %s\n",
			i+1, result.Title, result.URL, result.Content,
		))
	}

	if global.Env().IsDebug {
		log.Trace(util.MustToJSON(results))
	}

	return strings.Join(results, "\n---\n"), nil
}
