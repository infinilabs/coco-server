/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"encoding/json"
	"testing"

	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

func sectionsEvidence(t *testing.T, thin bool) util.MapStr {
	t.Helper()
	body := "这是一段足够长的正文内容，用来验证 auto-approve 的薄页下限判断逻辑是否正常工作。长度必须超过三百个字符的下限才算厚页，所以这里继续补充一些无意义但必要的中文文本。数据源连接器是一种用于从外部服务获取数据的工具，它定义了访问和拉取数据的具体方式，可以被注册并与搜索引擎集成。通过注册连接器，用户可以把远程系统中的文档定期同步进索引，再经由统一搜索入口进行混合检索与结果分组展示，从而让分散在多个系统里的信息获得一致的查询体验。此外正文还应当覆盖配置方式与常见限制，确保内容密度足以支撑发布门槛的判断。最后再补充一段关于检索行为的说明：混合检索会把关键词与语义两路召回融合排序，命中结果按数据源分组展示，用户可以通过快捷键在分组之间快速切换，亦可将某个来源固定为单源视图以查看更完整的结果列表。"
	if thin {
		body = "太短"
	}
	secs := []core.WikiSection{{ID: "s1", Heading: "概述", Body: body}}
	raw, err := json.Marshal(secs)
	if err != nil {
		t.Fatal(err)
	}
	return util.MapStr{"sections": raw}
}

func TestAutoApproveBodyChars_ShapeTolerant(t *testing.T) {
	thick := sectionsEvidence(t, false)
	thin := sectionsEvidence(t, true)

	if got := autoApproveBodyChars(&core.WikiGovernanceProposal{Evidence: thick}); got < autoApproveMinBodyChars {
		t.Fatalf("thick page counted %d, want >= %d", got, autoApproveMinBodyChars)
	}
	if got := autoApproveBodyChars(&core.WikiGovernanceProposal{Evidence: thin}); got >= autoApproveMinBodyChars {
		t.Fatalf("thin page counted %d, want < %d", got, autoApproveMinBodyChars)
	}

	// ES round-trip may decode RawMessage into a JSON string
	asString := thick
	switch v := thick["sections"].(type) {
	case json.RawMessage:
		asString["sections"] = string(v)
	case []byte:
		asString["sections"] = string(v)
	}
	if got := autoApproveBodyChars(&core.WikiGovernanceProposal{Evidence: asString}); got < autoApproveMinBodyChars {
		t.Fatalf("string-encoded evidence counted %d, want >= %d", got, autoApproveMinBodyChars)
	}

	// missing / undecodable evidence gates closed (0), never bypasses
	if got := autoApproveBodyChars(&core.WikiGovernanceProposal{}); got != 0 {
		t.Fatalf("empty evidence counted %d, want 0", got)
	}
	broken := util.MapStr{"sections": "not-json"}
	if got := autoApproveBodyChars(&core.WikiGovernanceProposal{Evidence: broken}); got != 0 {
		t.Fatalf("broken evidence counted %d, want 0", got)
	}
}
