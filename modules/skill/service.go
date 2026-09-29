/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package skill

import (
	"strings"

	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
)

// maxInjectedSkills bounds the prompt injection; plenty for curated sets
// while protecting the context window from runaway skill hoarding.
const maxInjectedSkills = 20

// GetEnabledSkills returns the enabled skills in prompt order.
func GetEnabledSkills() ([]core.Skill, error) {
	ctx := orm.NewContext()
	ctx.DirectAccess()

	skills := []core.Skill{}
	q := orm.Query{Conds: orm.And(orm.Eq("enabled", true)), Size: maxInjectedSkills}
	q.AddSort("sort_order", orm.ASC)
	err, _ := orm.SearchWithJSONMapper(&skills, &q)
	if err != nil {
		return nil, err
	}
	return skills, nil
}

// BuildSkillsSection renders the enabled skills as the Markdown block that
// gets appended to the assistant's system prompt. Empty string when no skill
// is enabled. Re-read per request: skill changes take effect on the next
// message without a restart, mirroring the console toggle latency.
//
// Prompt assembly must never break the reply pipeline — storage errors and
// even panics (e.g. the ORM backend not being registered in unit-test
// environments) degrade to "no skills" instead.
func BuildSkillsSection() string {
	return buildSkillsSection()
}

func buildSkillsSection() (section string) {
	defer func() {
		if r := recover(); r != nil {
			section = ""
		}
	}()

	skills, err := GetEnabledSkills()
	if err != nil {
		return ""
	}
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## 已激活的技能\n你同时具备以下专业能力，请在相关场景下主动运用：\n")
	for _, s := range skills {
		sb.WriteString("\n### ")
		sb.WriteString(s.Title)
		sb.WriteString("\n")
		sb.WriteString(s.Instructions)
		sb.WriteString("\n")
	}
	return sb.String()
}
