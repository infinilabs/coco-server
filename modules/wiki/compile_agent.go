/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	einoOpenAI "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	llmmodule "infini.sh/coco/modules/llm"
)

// The compile agent (DESIGN-compile-agent.md): the knowledge compiler
// reimagined as a continuously iterating EINO ReAct agent. Instead of the
// one-shot MAP→REDUCE→PLAN→WRITE pipeline, the agent assesses the KB's
// topic pages against the corpus with tools (search documents, read
// content, read the evolution log), then improves ONE page per iteration
// through a compile_patch governance proposal — the usual gate applies,
// the agent never writes directly.
//
// EINO boundary: this subsystem only. The assistant's deep think/research
// keep their langchaingo stack.

const (
	compileAgentTaskID          = "wiki-compile-agent"
	compileAgentDefaultInterval = "30m"
	compileAgentMaxSteps        = 24
	compileAgentMaxItersPerRun  = 3
	compileAgentReadLimit       = 6000
	compileAgentPageLimit       = 9000
	compileAgentTimeout         = 180 * time.Second
)

func init() {
	interval := strings.TrimSpace(os.Getenv("WIKI_COMPILE_AGENT_INTERVAL"))
	if interval == "" {
		interval = compileAgentDefaultInterval
	}
	switch strings.ToLower(interval) {
	case "off", "disabled", "0":
		return
	}
	task.RegisterScheduleTask(task.ScheduleTask{
		ID:          compileAgentTaskID,
		Group:       "wiki",
		Description: "Wiki 编译 Agent:EINO ReAct 循环持续评估并修缮专题页,产物走治理提案(DESIGN-compile-agent.md)",
		Interval:    interval,
		Singleton:   true,
		Task:        compileAgentSweep,
	})
}

func compileAgentSweep(ctx context.Context) {
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiKnowledgeBase{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(50))
	if err != nil || res == nil {
		return
	}
	kbs, _, derr := elastic.DecodeHits[core.WikiKnowledgeBase](res)
	if derr != nil {
		return
	}
	for i := range kbs {
		start := time.Now()
		report, err := runCompileAgent(octx, kbs[i].ID, compileAgentMaxItersPerRun)
		if err != nil {
			log.Warnf("wiki: compile agent failed for kb [%s]: %v", kbs[i].ID, err)
		} else if report != nil && (report.Patches > 0 || report.Iterations > 0) {
			log.Infof("wiki: compile agent kb [%s]: %d iterations, %d patches in %s",
				kbs[i].ID, report.Iterations, report.Patches, time.Since(start))
		}
	}
}

/* ---------------- model adapter ---------------- */

// resolveEinoChatModel adapts coco's default language model (an
// OpenAI-compatible provider) to an EINO tool-calling chat model. v1
// boundary: non-OpenAI-compatible providers (anthropic/gemini) fall back
// to the one-shot compile — the agent needs function calling.
func resolveEinoChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	modelID := llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	if modelID == nil {
		return nil, fmt.Errorf("no default language model configured")
	}
	provider, err := common.GetModelProvider(modelID.ProviderID)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, fmt.Errorf("model provider [%s] not found", modelID.ProviderID)
	}
	if apiType := strings.ToLower(provider.APIType); apiType != "" && apiType != "openai" {
		return nil, fmt.Errorf("compile agent needs an OpenAI-compatible provider, provider [%s] is %s", provider.Name, provider.APIType)
	}
	return einoOpenAI.NewChatModel(ctx, &einoOpenAI.ChatModelConfig{
		BaseURL: provider.BaseURL,
		APIKey:  provider.APIKey,
		Model:   modelID.ID,
		Timeout: compileAgentTimeout,
	})
}

/* ---------------- tools ---------------- */

// compileAgentTools carries the KB scope: every tool closure reads and
// proposes strictly inside one knowledge base.
type compileAgentTools struct {
	kb      core.WikiKnowledgeBase
	fileN   int // proposals actually filed this round
	noFiles int // propose attempts blocked by the open-twin guard
}

// pipeCtx converts the plain context EINO hands to tools into an orm
// context with the direct-read flags the pipeline-less path needs.
func pipeCtx(ctx context.Context) *orm.Context {
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	octx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	return octx
}

type docHit struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary,omitempty"`
}

type searchResults struct {
	Results []docHit `json:"results"`
}

func (t *compileAgentTools) searchDocuments(ctx context.Context, args struct {
	Query string `json:"query" jsonschema:"关键词查询词"`
	Size  int    `json:"size,omitempty" jsonschema:"返回条数,默认5,最大10"`
}) (*searchResults, error) {
	log.Infof("wiki: compile agent tool search_documents query=%r size=%d", args.Query, args.Size)
	size := args.Size
	if size <= 0 || size > 10 {
		size = 5
	}
	octx := pipeCtx(ctx)
	orm.WithModel(octx, &core.Document{})
	builder := orm.NewQuery().Size(size).
		Filter(orm.TermsQuery("source.id", t.kb.DatasourceIDs))
	if strings.TrimSpace(args.Query) != "" {
		builder.Must(orm.MatchQuery("combined_fulltext", args.Query))
	}
	builder.SortBy(orm.Sort{Field: "_score", SortType: "desc"})
	res, err := orm.SearchV2(octx, builder)
	if err != nil || res == nil {
		return &searchResults{}, nil
	}
	docs, _, derr := elastic.DecodeHits[core.Document](res)
	if derr != nil {
		return &searchResults{}, nil
	}
	out := &searchResults{Results: make([]docHit, 0, len(docs))}
	for i := range docs {
		out.Results = append(out.Results, docHit{
			ID:      docs[i].ID,
			Title:   docs[i].Title,
			Summary: truncateRunes(docs[i].Summary, 200),
		})
	}
	return out, nil
}

type documentBody struct {
	DocID   string `json:"doc_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	URL     string `json:"url,omitempty"`
}

func (t *compileAgentTools) readDocument(ctx context.Context, args struct {
	DocID string `json:"doc_id" jsonschema:"search_documents 返回的文档 id"`
}) (*documentBody, error) {
	log.Infof("wiki: compile agent tool read_document doc=%s", args.DocID)
	octx := pipeCtx(ctx)
	orm.WithModel(octx, &core.Document{})
	doc := core.Document{}
	doc.SetID(args.DocID)
	exists, err := orm.GetV2(octx, &doc)
	if err != nil || !exists {
		// models pass titles here often — fall back to a title match within
		// the KB's datasources before giving up
		res, serr := orm.SearchV2(octx, orm.NewQuery().Size(1).
			Filter(orm.TermsQuery("source.id", t.kb.DatasourceIDs), orm.TermQuery("title", args.DocID)))
		if serr == nil && res != nil {
			if hits, _, hderr := elastic.DecodeHits[core.Document](res); hderr == nil && len(hits) > 0 {
				doc = hits[0]
				exists = true
			}
		}
		if !exists {
			// a tool ERROR aborts the whole graph — return the miss as a
			// result so the model can self-correct
			return &documentBody{DocID: args.DocID, Content: fmt.Sprintf("document %q not found; use search_documents to list valid ids", args.DocID)}, nil
		}
	}
	inScope := false
	for _, dsID := range t.kb.DatasourceIDs {
		if doc.Source.ID == dsID {
			inScope = true
			break
		}
	}
	if !inScope {
		return &documentBody{DocID: doc.ID, Content: fmt.Sprintf("document %q belongs to datasource [%s], outside this knowledge base", doc.Title, doc.Source.ID)}, nil
	}
	return &documentBody{
		DocID:   doc.ID,
		Title:   doc.Title,
		Content: truncateRunes(doc.Content, compileAgentReadLimit),
		URL:     doc.URL,
	}, nil
}

type topicPageRef struct {
	Title   string `json:"title"`
	Summary string `json:"summary,omitempty"`
	Updated string `json:"updated,omitempty"`
}

type topicPageList struct {
	Pages []topicPageRef `json:"pages"`
}

// findTopicPages lists a KB's topic pages.
func findTopicPages(ctx *orm.Context, kbID string, size int) []core.WikiArticle {
	orm.WithModel(ctx, &core.WikiArticle{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(size).
		Filter(orm.TermQuery("kb_id", kbID), orm.TermQuery("page_type", core.WikiPageTypeTopic)))
	if err != nil || res == nil {
		return nil
	}
	pages, _, derr := elastic.DecodeHits[core.WikiArticle](res)
	if derr != nil {
		return nil
	}
	return pages
}

func (t *compileAgentTools) listTopicPages(ctx context.Context, args struct{}) (*topicPageList, error) {
	log.Infof("wiki: compile agent tool list_topic_pages")
	pages := findTopicPages(pipeCtx(ctx), t.kb.ID, 100)
	out := &topicPageList{Pages: make([]topicPageRef, 0, len(pages))}
	for i := range pages {
		updated := ""
		if pages[i].Updated != nil {
			updated = pages[i].Updated.Format("2006-01-02 15:04")
		}
		out.Pages = append(out.Pages, topicPageRef{
			Title:   pages[i].Title,
			Summary: truncateRunes(pages[i].Summary, 100),
			Updated: updated,
		})
	}
	return out, nil
}

type topicPageBody struct {
	Title   string   `json:"title"`
	Summary string   `json:"summary,omitempty"`
	Content string   `json:"content"`
	Outline []string `json:"outline,omitempty"`
}

func (t *compileAgentTools) readTopicPage(ctx context.Context, args struct {
	Title string `json:"title" jsonschema:"专题页标题"`
}) (*topicPageBody, error) {
	log.Infof("wiki: compile agent tool read_topic_page title=%q", args.Title)
	octx := pipeCtx(ctx)
	orm.WithModel(octx, &core.WikiArticle{})
	article := findArticleByTitle(octx, t.kb.ID, args.Title)
	if article == nil {
		return &topicPageBody{Title: args.Title, Content: fmt.Sprintf("no topic page titled %q — call list_topic_pages for exact titles", args.Title)}, nil
	}
	out := &topicPageBody{
		Title:   article.Title,
		Summary: article.Summary,
		Content: truncateRunes(article.Content, compileAgentPageLimit),
	}
	for _, s := range article.Outline {
		out.Outline = append(out.Outline, s.Heading)
	}
	return out, nil
}

type entityRef struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

type entityList struct {
	Entities   []entityRef `json:"entities"`
	TotalCount int         `json:"total_count"`
}

func (t *compileAgentTools) listEntities(ctx context.Context, args struct{}) (*entityList, error) {
	log.Infof("wiki: compile agent tool list_entities")
	octx := pipeCtx(ctx)
	orm.WithModel(octx, &core.WikiEntity{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(60).
		Filter(orm.TermQuery("kb_id", t.kb.ID)))
	if err != nil || res == nil {
		return &entityList{}, nil
	}
	entities, total, derr := elastic.DecodeHits[core.WikiEntity](res)
	if derr != nil {
		return &entityList{}, nil
	}
	out := &entityList{Entities: make([]entityRef, 0, len(entities)), TotalCount: int(total)}
	for i := range entities {
		out.Entities = append(out.Entities, entityRef{Name: entities[i].Name, Type: entities[i].Type})
	}
	return out, nil
}

type logLine struct {
	Kind    string `json:"kind"`
	Summary string `json:"summary"`
	Created string `json:"created,omitempty"`
}

type evolutionLogView struct {
	Entries []logLine `json:"entries"`
}

func (t *compileAgentTools) readEvolutionLog(ctx context.Context, args struct{}) (*evolutionLogView, error) {
	log.Infof("wiki: compile agent tool read_evolution_log")
	entries := findEvolutionEntries(pipeCtx(ctx), t.kb.ID, 20)
	out := &evolutionLogView{Entries: make([]logLine, 0, len(entries))}
	for i := range entries {
		out.Entries = append(out.Entries, logLine{
			Kind:    entries[i].Kind,
			Summary: truncateRunes(entries[i].Summary, 120),
			Created: entries[i].Created.Format("01-02 15:04"),
		})
	}
	return out, nil
}

type patchFileResult struct {
	Filed  bool   `json:"filed"`
	Reason string `json:"reason,omitempty"`
}

func (t *compileAgentTools) proposePagePatch(ctx context.Context, args struct {
	Title     string `json:"title" jsonschema:"要修缮的专题页标题(必须已存在)"`
	Rationale string `json:"rationale" jsonschema:"为什么这样改,引用哪些证据"`
	Content   string `json:"content" jsonschema:"修订后的完整页面 Markdown 正文(整页替换,不是片段)"`
}) (*patchFileResult, error) {
	log.Infof("wiki: compile agent tool propose_page_patch title=%q content=%d chars", args.Title, len([]rune(args.Content)))
	octx := pipeCtx(ctx)
	orm.WithModel(octx, &core.WikiArticle{})
	title := strings.TrimSpace(args.Title)
	content := strings.TrimSpace(args.Content)
	if title == "" || len([]rune(content)) < 200 {
		return &patchFileResult{Filed: false, Reason: "标题为空或正文不足 200 字"}, nil
	}
	article := findArticleByTitle(octx, t.kb.ID, title)
	if article == nil {
		return &patchFileResult{Filed: false, Reason: fmt.Sprintf("页面 %q 不存在——只允许修缮已有专题页", title)}, nil
	}
	if openPatternTwin(octx, t.kb.ID, core.WikiGovernanceCompilePatch, title) {
		t.noFiles++
		return &patchFileResult{Filed: false, Reason: "该页已有待审的修订提案，本轮跳过"}, nil
	}
	orm.WithModel(octx, &core.WikiGovernanceProposal{})
	proposal := &core.WikiGovernanceProposal{
		KbID:         t.kb.ID,
		ArticleID:    article.ID,
		ArticleTitle: article.Title,
		Type:         core.WikiGovernanceCompilePatch,
		Status:       core.WikiGovernanceOpen,
		Reason:       truncateRunes(args.Rationale, 200),
		Evidence: util.MapStr{
			"content":   content,
			"rationale": truncateRunes(args.Rationale, 400),
		},
	}
	if err := orm.Create(octx, proposal); err != nil {
		log.Warnf("wiki: compile agent propose [%s] failed: %v", title, err)
		return &patchFileResult{Filed: false, Reason: err.Error()}, nil
	}
	t.fileN++
	log.Infof("wiki: compile agent filed compile_patch [%s] for kb [%s] (%d chars)", title, t.kb.ID, len([]rune(content)))
	// the model otherwise keeps exploring until the step budget kills the
	// whole run — the proposal is in, the round is DONE
	return &patchFileResult{Filed: true, Reason: "提案已登记。本轮任务完成：立即输出最终 JSON 结论（action=patch_proposed），不要再调用任何工具。"}, nil
}

func (t *compileAgentTools) einoTools(ctx context.Context, kbID string) ([]tool.BaseTool, error) {
	search, err := utils.InferTool("search_documents", "按关键词检索本知识库绑定数据源里的文档,返回 id/标题/摘要", t.searchDocuments)
	if err != nil {
		return nil, err
	}
	readDoc, err := utils.InferTool("read_document", "读取一篇文档的正文(截断),用于核对引用与补充证据", t.readDocument)
	if err != nil {
		return nil, err
	}
	listPages, err := utils.InferTool("list_topic_pages", "列出本知识库全部专题页(标题+摘要+更新时间)", t.listTopicPages)
	if err != nil {
		return nil, err
	}
	readPage, err := utils.InferTool("read_topic_page", "读取一个专题页的现有全文与大纲", t.readTopicPage)
	if err != nil {
		return nil, err
	}
	listEnt, err := utils.InferTool("list_entities", "列出现有本体实体", t.listEntities)
	if err != nil {
		return nil, err
	}
	readLog, err := utils.InferTool("read_evolution_log", "读取知识库进化日志(含被拒记录),避免重复提交", t.readEvolutionLog)
	if err != nil {
		return nil, err
	}
	propose, err := utils.InferTool("propose_page_patch", "提交一个专题页的整页修订提案(走治理闸门),每轮至多提交一次", t.proposePagePatch)
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{search, readDoc, listPages, readPage, listEnt, readLog, propose}, nil
}

/* ---------------- agent loop ---------------- */

type compileAgentRound struct {
	Action    string `json:"action"`
	Page      string `json:"page,omitempty"`
	Rationale string `json:"rationale,omitempty"`
	Patches   int    `json:"patches"`
}

type compileAgentReport struct {
	Iterations int      `json:"iterations"`
	Patches    int      `json:"patches"`
	NoAction   bool     `json:"no_action"`
	Notes      []string `json:"notes,omitempty"`
}

const compileAgentPersona = `你是知识库的持续编译维护者。你的职责是让专题页与底层语料保持一致：内容过时、证据薄弱、标题与内容不符的页面都要修缮。

工作规则：
1. 工具调用预算只有 8 次，超预算任务即告失败——按此规划步数
2. 每轮只改进一个页面（原子迭代）：list_topic_pages 一次通览 → read_topic_page 深入 1-2 个候选 → search_documents/read_document 取证 ≤3 次 → propose_page_patch 一次提交
3. 修订必须以工具取回的真实内容为据，正文里用 [Doc: 文档标题] 标注来源；禁止编造
4. 提交前自查：内容是否答标题？每个关键主张是否有出处？比现有版本好吗？没把握就 no_action
5. propose_page_patch 提交的是整页替换正文（Markdown，## 小节结构，无一级标题）
6. 提交完成后立即结束，最终回答必须是纯 JSON（不要代码块不要解释）：
{"action":"patch_proposed" 或 "no_action","page":"页面标题","rationale":"一句话理由"}`

// runCompileAgentOnce runs ONE agent iteration: pick a page, gather
// evidence, file at most one compile_patch proposal (or no_action).
func runCompileAgentOnce(ctx context.Context, kb core.WikiKnowledgeBase, tools *compileAgentTools) (*compileAgentRound, error) {
	cm, err := resolveEinoChatModel(ctx)
	if err != nil {
		return nil, err
	}
	einoTools, err := tools.einoTools(ctx, kb.ID)
	if err != nil {
		return nil, err
	}
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: cm,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: einoTools},
		MaxStep:          compileAgentMaxSteps,
		MessageModifier:  react.NewPersonaModifier(compileAgentPersona),
	})
	if err != nil {
		return nil, fmt.Errorf("create react agent: %w", err)
	}

	userPrompt := fmt.Sprintf(`知识库「%s」（数据源：%s）。请按你的工作规则完成一轮维护：评估专题页 → 取证 → 至多提交一个 propose_page_patch → 给出 JSON 结论。`, kb.Name, strings.Join(kb.DatasourceIDs, ", "))

	cctx, cancel := context.WithTimeout(ctx, compileAgentTimeout)
	defer cancel()
	msg, err := agent.Generate(cctx, []*schema.Message{schema.UserMessage(userPrompt)})
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, fmt.Errorf("agent returned no message")
	}
	prose := msg.Content
	if i := strings.Index(prose, "</think>"); i >= 0 {
		prose = prose[i+len("</think>"):]
	}
	prose = strings.TrimSpace(prose)
	prose = strings.TrimPrefix(prose, "```json")
	prose = strings.TrimPrefix(prose, "```")
	prose = strings.TrimSpace(prose)
	if i := strings.LastIndex(prose, "```"); i >= 0 {
		prose = strings.TrimSpace(prose[:i])
	}
	var verdict compileAgentRound
	if jerr := json.Unmarshal([]byte(prose), &verdict); jerr != nil {
		// the model talked instead of returning JSON — treat as no_action
		// with its own words as the rationale
		return &compileAgentRound{Action: "no_action", Rationale: truncateRunes(prose, 200)}, nil
	}
	verdict.Patches = tools.fileN
	return &verdict, nil
}

// runCompileAgent drives up to maxIters iterations for one KB. The loop
// stops early on no_action or when the agent stops filing patches.
func runCompileAgent(parent *orm.Context, kbID string, maxIters int) (*compileAgentReport, error) {
	if maxIters <= 0 {
		maxIters = 1
	}
	kb := core.WikiKnowledgeBase{}
	kb.SetID(kbID)
	kctx := orm.NewContextWithParent(parent)
	kctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(kctx, &core.WikiKnowledgeBase{})
	exists, err := orm.GetV2(kctx, &kb)
	if err != nil || !exists {
		return nil, fmt.Errorf("knowledge base %s not found", kbID)
	}

	report := &compileAgentReport{}
	ctx := orm.NewContextWithParent(parent)
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)

	// run-wide tool state: a proposal filed inside an iteration that later
	// trips the step budget is STILL filed — the count must survive the run
	tools := &compileAgentTools{kb: kb}

	for i := 0; i < maxIters; i++ {
		round, rerr := runCompileAgentOnce(ctx, kb, tools)
		if rerr != nil {
			// a step-budget overrun after a filed patch is a soft failure:
			// the proposal stands, the next run continues from the log
			report.Notes = append(report.Notes, truncateRunes(rerr.Error(), 160))
			break
		}
		report.Iterations++
		report.NoAction = strings.EqualFold(round.Action, "no_action")
		if round.Page != "" || round.Rationale != "" {
			note := round.Page
			if note == "" {
				note = truncateRunes(round.Rationale, 100)
			}
			report.Notes = append(report.Notes, fmt.Sprintf("迭代%d: %s — %s", i+1, round.Action, note))
		}
		if report.NoAction || tools.fileN == 0 {
			break
		}
	}
	report.Patches = tools.fileN

	AppendEvolutionEntry(parent, kbID, core.WikiEvolutionCompileAgent, "compile-agent",
		"kb", kbID,
		fmt.Sprintf("编译 Agent：%d 轮迭代，%d 个修订提案", report.Iterations, report.Patches),
		util.MapStr{
			"iterations": report.Iterations,
			"patches":    report.Patches,
			"no_action":  report.NoAction,
			"notes":      report.Notes,
		})
	return report, nil
}

/* ---------------- manual trigger ---------------- */

func (h *APIHandler) runCompileAgentHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	report, err := runCompileAgent(ctx, kbID, compileAgentMaxItersPerRun)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, report)
}
