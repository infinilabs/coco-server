/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"net/http"
	"strings"

	log "github.com/cihub/seelog"

	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/service"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/api/crud"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

/* ---------------- scenario templates ---------------- */

const templateResource = "assistant_template"

func registerTemplateCRUD() {
	readPermission := security.GetSimplePermission(Category, templateResource, string(security.Read))
	searchPermission := security.GetSimplePermission(Category, templateResource, string(security.Search))
	security.GetOrInitPermissionKeys(readPermission, searchPermission)
	// templates are user-facing presets, readable by every chat user
	security.RegisterPermissionsToRole(core.WidgetRole, readPermission, searchPermission)

	crud.RegisterCRUD[core.AssistantTemplate](crud.Config[core.AssistantTemplate]{
		// own prefix: a /assistant/template/:id child would conflict with the
		// /assistant/:id wildcard in the router tree
		Prefix:             "/assistant-template",
		Resource:           templateResource,
		Permission:         simpleTemplatePermissionFn(),
		DefaultQueryFields: []string{"title", "description"},
		MCP:                true,
		MCPDescs: map[string]string{
			crud.ActionSearch: "List assistant scenario templates (support/sales/HR/IT presets); instantiate one via instantiate_assistant_template",
			crud.ActionRead:   "Get one assistant template by id",
		},
		// seeds are the only writer; templates are instantiated, never edited
		SkipActions: []string{crud.ActionCreate, crud.ActionUpdate, crud.ActionDelete},
	})
}

func simpleTemplatePermissionFn() func(action string) api.PermissionKey {
	return func(action string) api.PermissionKey {
		return security.GetSimplePermission(Category, templateResource, action)
	}
}

/* ---------------- POST /assistant/template/:id/_instantiate ---------------- */

// instantiateTemplate copies a template into a live assistant (never builtin),
// optionally binding it to a knowledge base: the assistant's data scope
// becomes the KB's datasources and the KB records the assistant back — the
// previously unwired WikiKb.AssistantID field becomes a real binding.
func (h *APIHandler) instantiateTemplate(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	templateID := ps.MustGetParameter("id")

	var body struct {
		Name string `json:"name"`
		KbID string `json:"kb_id"`
	}
	if req.ContentLength != 0 {
		if err := h.DecodeJSON(req, &body); err != nil {
			h.Error400(w, err.Error())
			return
		}
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.AssistantTemplate{})

	var template core.AssistantTemplate
	template.SetID(templateID)
	exists, err := orm.GetV2(ctx, &template)
	if err != nil {
		if !strings.Contains(err.Error(), "record not found") {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		exists = false // the sqlite backend reports a miss as an error
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, templateID)
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = template.Title
	}

	assistant := &core.Assistant{
		Name:         name,
		Description:  template.Description,
		Icon:         template.Icon,
		Type:         core.AssistantTypeSimple,
		Category:     template.Category,
		Enabled:      true,
		Builtin:      false,
		ToolsConfig:  template.ToolsConfig,
		MCPConfig:    template.MCPConfig,
		ChatSettings: template.ChatSettings,
		RolePrompt:   template.RolePrompt,
	}

	var kb *core.WikiKnowledgeBase
	if body.KbID != "" {
		ormCtx := orm.NewContextWithParent(req.Context())
		ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
		ormCtx.Set(orm.DirectWriteWithoutPermissionCheck, true)
		orm.WithModel(ormCtx, &core.WikiKnowledgeBase{})

		var loaded core.WikiKnowledgeBase
		loaded.SetID(body.KbID)
		kbExists, err := orm.GetV2(ormCtx, &loaded)
		if err != nil {
			if !strings.Contains(err.Error(), "record not found") {
				h.WriteError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			kbExists = false // the sqlite backend reports a miss as an error
		}
		if !kbExists {
			h.WriteOpRecordNotFoundJSON(w, body.KbID)
			return
		}
		kb = &loaded
		if len(kb.DatasourceIDs) > 0 {
			assistant.Datasource = core.DatasourceConfig{
				Enabled: true,
				IDs:     append([]string{}, kb.DatasourceIDs...),
				Visible: true,
			}
		}
	}

	createCtx := orm.NewContextWithParent(req.Context())
	createCtx.Refresh = orm.WaitForRefresh
	orm.WithModel(createCtx, assistant)
	if err := orm.Create(createCtx, assistant); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	service.ClearAssistantsCache()

	if kb != nil {
		// write the reverse binding so the KB face can offer "ask this KB"
		ormCtx := orm.NewContextWithParent(req.Context())
		ormCtx.Set(orm.DirectWriteWithoutPermissionCheck, true)
		ormCtx.Refresh = orm.WaitForRefresh
		orm.WithModel(ormCtx, &core.WikiKnowledgeBase{})

		kb.AssistantID = assistant.ID
		if err := orm.Update(ormCtx, kb); err != nil {
			log.Warnf("assistant: template kb writeback failed: %v", err)
		}
	}

	h.WriteCreatedOKJSON(w, assistant.ID)
}

/* ---------------- seeds ---------------- */

// builtinTemplateSeeds ship five scenario presets. Like skill seeds they are
// inserted once by Name and never overwritten — local edits survive reboots.
var builtinTemplateSeeds = []core.AssistantTemplate{
	{
		Name:        "customer-support",
		Title:       "客服助手",
		Category:    "support",
		Icon:        "🎧",
		SortOrder:   10,
		Builtin:     true,
		Description: "面向客户咨询的一线应答:基于产品文档与工单知识作答,语气友好, escalated 场景给出转人工建议",
		RolePrompt: `你是一名称职的企业产品客服代表。回答客户问题时:
1. 优先依据内部知识库检索结果作答,引用具体文档;检索不到的保修政策、价格承诺不要编造,明确建议转人工。
2. 语气友好、口语化,先给结论再给步骤;操作指引用有序列表,一步一行。
3. 涉及退款、投诉、账号安全等敏感场景,给出标准流程并提示需要的凭证,不擅自承诺赔偿。
4. 回答末尾主动确认问题是否解决,并给一条最相关的后续建议。`,
		SuggestedQuestions: []string{"退款政策是怎样的?", "如何重置密码?", "订单一直显示处理中怎么办?"},
	},
	{
		Name:        "sales-presales",
		Title:       "销售售前顾问",
		Category:    "sales",
		Icon:        "💼",
		SortOrder:   20,
		Builtin:     true,
		Description: "售前咨询辅助:基于产品资料对比方案、准备话术、起草跟进邮件,不承诺未经审批的价格",
		RolePrompt: `你是一名 B2B 售前顾问的助理。支持销售同事时:
1. 产品能力、规格、案例一律以内部资料为准,注明出处;资料未覆盖的能力差异要明说。
2. 对比竞品时保持客观,只陈述有据可查的差异,不贬损竞品。
3. 起草邮件/话术时先给一版完整草稿,再附 2-3 条可选的语气调整建议。
4. 价格、折扣、交付承诺只能引用官方价格文档;没有文档就写"[价格待确认]",提醒销售走审批。`,
		SuggestedQuestions: []string{"帮我对比两个方案给客户", "给犹豫的客户写一封跟进邮件", "这个产品的典型客户案例有哪些?"},
	},
	{
		Name:        "it-helpdesk",
		Title:       "IT 服务台",
		Category:    "it",
		Icon:        "🛠️",
		SortOrder:   30,
		Builtin:     true,
		Description: "内部 IT 支持:VPN/邮箱/账号/软件安装等常见问题自助答疑,按知识库 SOP 给出步骤",
		RolePrompt: `你是企业 IT 服务台助手。处理内部同事的技术求助时:
1. 严格按内部 IT 知识库的 SOP 回答:步骤编号、命令用代码块、注明适用系统版本。
2. 涉及账号权限、安全策略的操作,提示需要走工单/审批的部分,不要引导绕过安全策略。
3. SOP 未覆盖的问题,给出通用排查思路并明确"内部文档未覆盖",建议提工单。
4. 高危操作(重装、清数据、改注册表)必须先提示备份与影响范围。`,
		SuggestedQuestions: []string{"VPN 连不上怎么排查?", "如何申请软件安装权限?", "邮箱满了怎么清理?"},
	},
	{
		Name:        "hr-policy",
		Title:       "HR 政策问答",
		Category:    "hr",
		Icon:        "📋",
		SortOrder:   40,
		Builtin:     true,
		Description: "员工制度咨询:考勤、假期、报销、福利等按公司制度文件作答,隐私问题指引正式渠道",
		RolePrompt: `你是企业 HR 政策问答助手。回答员工关于制度的提问时:
1. 只依据公司制度文件作答并注明文件名称;文件没有的规定就说"制度未明确",建议咨询 HR 专员。
2. 涉及个人 case(我的工资/我的假余额)时,指引到正式系统或 HR 渠道,不猜测个人数据。
3. 涉及劳动法规的口径以制度文件引用为准,不提供法律意见。
4. 回答保持中性、简洁,避免对制度本身作评价。`,
		SuggestedQuestions: []string{"年假怎么折算?", "报销流程和时限?", "试用期有几天病假?"},
	},
	{
		Name:        "meeting-recap",
		Title:       "会议纪要助手",
		Category:    "productivity",
		Icon:        "📝",
		SortOrder:   50,
		Builtin:     true,
		Description: "把会议记录/速记整理成结构化纪要:决议、待办、责任人、时限,并沉淀入知识库草稿",
		RolePrompt: `你是会议纪要整理助手。把用户提供的会议速记或录音转写整理为:
1. 结构:会议主题 → 关键决议(编号) → 待办事项(表格:事项/责任人/时限) → 遗留问题。
2. 只整理已说的内容,不补撰未讨论的决议;不确定的责任人标注"[待确认]"。
3. 口语去除、保留数字与日期原样;篇幅不超过原文 1/2。
4. 用户确认后可提示:可用"保存到知识库"把纪要沉淀为团队知识。`,
		SuggestedQuestions: []string{"帮我把这段速记整理成纪要", "从转写里提取所有待办和责任人", "上周的决议有哪些还没闭环?"},
	},
}

// seedBuiltinTemplates mirrors skill seeding: insert-by-Name, log-not-fatal.
func seedBuiltinTemplates() {
	ctx := orm.NewContext()
	ctx.DirectAccess()

	existing := []core.AssistantTemplate{}
	q := orm.Query{Conds: orm.And(orm.Eq("builtin", true)), Size: 100}
	if err, _ := orm.SearchWithJSONMapper(&existing, &q); err != nil {
		log.Errorf("assistant: load builtin templates for seeding failed: %v", err)
		return
	}

	known := util.MapStr{}
	for _, t := range existing {
		known[t.Name] = true
	}

	created := 0
	for _, seed := range builtinTemplateSeeds {
		if known[seed.Name] == true {
			continue
		}
		obj := seed
		if err := orm.Create(ctx, &obj); err != nil {
			log.Errorf("assistant: seed template %s failed: %v", seed.Name, err)
			continue
		}
		created++
	}
	if created > 0 {
		log.Infof("assistant: seeded %d builtin scenario templates", created)
	}
}

// scheduleSeedTemplates runs after full setup so the ORM backend is ready.
func scheduleSeedTemplates() {
	global.RegisterFuncAfterSetup(func() {
		seedBuiltinTemplates()
	})
}
