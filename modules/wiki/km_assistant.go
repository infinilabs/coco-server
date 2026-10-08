/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"context"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
)

/* The builtin KM assistant: a ready-to-bind persona for the wiki AI pipeline.
 * Bind it to a knowledge base (settings → AI 智能体) and its answering model
 * drives generation/editing while this role prompt steers drafting. The
 * answering model is intentionally empty — generation falls back to the
 * server default language model. Seeded once at setup, idempotent. */

const builtinKmAssistantID = "builtin-wiki-km"

const builtinKmRolePrompt = `You curate a team knowledge base from indexed source documents.
- Structure pages as markdown: lead with a definition, then focused "##" sections; prefer concise lists over prose walls.
- Ground every claim in the provided source documents and cite inline as [1], [2] — never present uncited material as fact and never invent sources.
- Reuse existing [[type:Name]] wikilinks where the knowledge-base vocabulary has them.
- Match the language of the source material; stay factual and concise, no marketing tone.
- Everything you produce is a draft for human review — flag uncertainties explicitly instead of guessing.`

func scheduleSeedBuiltinKmAssistant() {
	global.RegisterFuncAfterSetup(func() {
		seedBuiltinKmAssistant()
	})
}

func seedBuiltinKmAssistant() {
	ctx := orm.NewContextWithParent(context.Background())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.Assistant{})
	var existing core.Assistant
	existing.SetID(builtinKmAssistantID)
	exists, err := orm.GetV2(ctx, &existing)
	if err != nil {
		log.Errorf("wiki: check builtin KM assistant: %v", err)
		return
	}
	if exists {
		return
	}

	assistant := core.Assistant{
		Name:        "知识管理助手",
		Description: "Builtin persona for wiki AI generation/editing: citation-backed, draft-only knowledge management. Bind it to a knowledge base to steer AI output.",
		Icon:        "📓",
		Type:        "simple",
		Category:    "processing",
		Enabled:     true,
		Builtin:     true,
		RolePrompt:  builtinKmRolePrompt,
	}
	assistant.SetID(builtinKmAssistantID)

	createCtx := orm.NewContextWithParent(context.Background())
	createCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	createCtx.Refresh = orm.WaitForRefresh
	if err := orm.Create(createCtx, &assistant); err != nil {
		log.Errorf("wiki: seed builtin KM assistant: %v", err)
		return
	}
	log.Info("wiki: seeded builtin KM assistant (bind it in KB settings → AI 智能体)")
}
