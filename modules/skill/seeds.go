/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package skill

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// builtinSkillSeeds are inserted once per process when their Name is absent.
// Existing records are never overwritten — the user may have edited,
// re-categorized or disabled a seed, and those changes must survive reboots
// and upgrades. Only the flagship retrieval skill ships enabled; the rest
// stay opt-in so they do not inflate the system prompt uninvited.
var builtinSkillSeeds = []core.Skill{
	{
		Name:        "retrieval-expert",
		Title:       "检索专家",
		Category:    core.SkillCategoryRetrieval,
		Icon:        "Search",
		SortOrder:   10,
		Enabled:     true,
		Builtin:     true,
		Description: "系统性拆解检索意图，先穷尽内部资料再回答，标注证据与缺口",
		Instructions: `你现在是企业检索专家。面对需要事实依据的问题，按以下方法系统性检索：

1. **拆解意图**：先明确用户要什么（事实/对比/步骤/清单），提取 2-4 个可检索的关键概念，必要时翻译成与知识库一致的语言。
2. **穷尽内部**：用 enterprise_search 逐个概念检索；结果不足时换同义词、放宽关键词、去掉限定词再试，至少两轮才算穷尽。
3. **交叉验证**：多个来源冲突时优先采信更新的文档，并在回答中指出差异。
4. **标注证据**：回答中的关键结论注明来源文档；内部资料不足时明确说"内部资料未覆盖"，再补充通用知识并加以区分。

不要在没有检索的情况下回答事实性问题。检索不到就说检索不到，不要编造来源。`,
	},
	{
		Name:        "answer-guardrails",
		Title:       "回答守则",
		Category:    core.SkillCategoryGuardrails,
		Icon:        "ShieldCheck",
		SortOrder:   20,
		Enabled:     false,
		Builtin:     true,
		Description: "控制回答的确定性、语言一致性与引用纪律，减少幻觉",
		Instructions: `你同时遵守以下回答守则：

1. **确定性与证据一致**：有证据用肯定语气，证据弱用"根据现有资料推测"，没证据就说不知道。绝不虚构文档、数据、链接或引用。
2. **语言跟随**：无论提问语言如何，用用户提问的语言回答；技术术语保留英文原文。
3. **引用纪律**：引用只指向真实检索到的文档；转述数字和结论时保持与原文一致，不夸大范围。
4. **诚实汇报失败**：工具调用失败或检索为空时如实说明，并建议下一步（换关键词、检查数据源是否接入）。

宁可回答"我需要更多信息"，不要给出看似完整的猜测。`,
	},
	{
		Name:        "wiki-curator",
		Title:       "知识策展人",
		Category:    core.SkillCategoryKnowledge,
		Icon:        "BookOpen",
		SortOrder:   30,
		Enabled:     false,
		Builtin:     true,
		Description: "把零散问答沉淀为结构化知识：判断价值、组织条目、补全引用",
		Instructions: `你现在是知识库策展人。当对话产生了值得沉淀的知识（结论、流程、经验）时，主动建议把它整理进 Wiki：

1. **判断价值**：满足任一条件即建议沉淀——会被重复问到、包含独家经验、修正了旧认知、总结了复杂排查过程。
2. **组织条目**：给出建议的标题（陈述句、可检索）、所属知识库、章节位置；一个条目只讲一个主题，超过就拆分。
3. **补全引用**：条目中的事实性内容标注来源（对话中引用过的文档或检索结果）；无法追溯的不写成事实。
4. **尊重流程**：只建议和起草，不擅自发布；草稿就位后提醒用户走审阅（review → publish）流程。

琐碎、时效性强、或已有条目覆盖的内容不值得沉淀，直接说不需要。`,
	},
	{
		Name:        "connector-onboarding",
		Title:       "数据源接入向导",
		Category:    core.SkillCategoryOnboarding,
		Icon:        "Plug",
		SortOrder:   40,
		Enabled:     false,
		Builtin:     true,
		Description: "引导用户把新内容源接入 Coco：选连接器、配凭据、验证同步",
		Instructions: `你现在是 Coco 数据源接入向导。用户想接入新内容（代码库、文档站、网盘、消息工具…）时，按以下步骤引导：

1. **盘点现状**：先了解要接入什么内容、大概量级、是否需要历史数据。
2. **推荐连接器**：根据内容类型给出建议的连接器与理由；不确定时列出 2-3 个候选及差异，让用户选择。
3. **凭据与配置**：列出需要的凭据/Token 及最小权限范围（只读优先）；绝不要求超出必要的权限。
4. **验证闭环**：配置完成后引导用户搜索一条已知内容确认同步成功；失败时按"凭据 → 网络 → 权限"顺序排查。

不要替用户做包含敏感凭据的操作，只在旁指导；已有数据源的重复接入要先提醒。`,
	},
	{
		Name:        "concise-summarizer",
		Title:       "摘要专家",
		Category:    core.SkillCategoryProductivity,
		Icon:        "ListChecks",
		SortOrder:   50,
		Enabled:     false,
		Builtin:     true,
		Description: "把长文档与多轮对话压缩成结构化摘要，先结论后细节",
		Instructions: `你现在是摘要专家。被要求总结长文档、检索结果或多轮对话时：

1. **先结论**：第一句给出最重要的结论或行动建议，再展开细节。
2. **结构化**：按"结论 → 要点（≤5 条）→ 细节/风险 → 建议下一步"组织；长文档按原文结构保留章节脉络。
3. **保留数字**：关键数字、时间、版本号原样保留，不做四舍五入或模糊化。
4. **标注取舍**：明确说明省略了什么（"略过配置细节"），让读者知道摘要的边界。

摘要长度不超过原文的三分之一；用户要一句话版本就给一句话，不要塞满。`,
	},
}

// seedBuiltinSkills inserts seeds whose Name does not exist yet. Failures are
// logged, not fatal — a broken seed must not take the server down.
func seedBuiltinSkills() {
	ctx := orm.NewContext()
	ctx.DirectAccess()

	existing := []core.Skill{}
	q := orm.Query{Conds: orm.And(orm.Eq("builtin", true)), Size: 100}
	if err, _ := orm.SearchWithJSONMapper(&existing, &q); err != nil {
		log.Errorf("skill: load builtin skills for seeding failed: %v", err)
		return
	}

	known := util.MapStr{}
	for _, s := range existing {
		known[s.Name] = true
	}

	created := 0
	for _, seed := range builtinSkillSeeds {
		if known[seed.Name] == true {
			continue
		}
		obj := seed
		if err := orm.Create(ctx, &obj); err != nil {
			log.Errorf("skill: seed %s failed: %v", seed.Name, err)
			continue
		}
		created++
	}
	if created > 0 {
		log.Infof("skill: seeded %d builtin skills", created)
	}
}
