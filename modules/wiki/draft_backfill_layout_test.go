package wiki

import (
	"strings"
	"testing"

	"infini.sh/coco/core"
)

func TestComposeEntityContentWithTypeLayout(t *testing.T) {
	entity := core.WikiEntity{
		Name: "美心飲食禮券$20", Type: "product",
		Aliases: []string{"$20禮券"},
		Properties: map[string]interface{}{
			"face_value": "$20",
			"category":   "餐饮礼券",
		},
		Sources: []core.WikiSourceReference{{DocID: "d1", Title: "products.json", Excerpt: "physical group dining gift voucher"}},
		Relations: []core.WikiEntityRelation{
			{TargetID: "t1", TargetType: "organization", Relation: "issued_by"},
		},
	}
	layout := []OntologyPageSection{
		{Key: "properties", Title: "产品档案"},
		{Key: "sources", Title: "来源引用"},
		{Key: "relations:issued_by", Title: "发行方"},
		{Key: "notes", Title: "备注"},
		{Key: "aliases", Title: "别名"},
	}
	got := composeEntityContent(&entity, layout, map[string]string{"t1": "Maxims Group"})

	for _, want := range []string{
		"## 产品档案", "**face_value**: $20",
		"## 来源引用", "physical group dining gift voucher",
		"## 发行方", "[[organization:Maxims Group]]（issued_by）",
		"## 备注", "（待补充）",
		"## 别名", "$20禮券",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("page missing %q\n---\n%s", want, got)
		}
	}
	// section order follows the layout: 产品档案 before 来源引用
	if strings.Index(got, "## 产品档案") > strings.Index(got, "## 来源引用") {
		t.Errorf("sections must follow the declared order")
	}
	// canonical layout (no declared sections) keeps the old shape
	canonical := composeEntityContent(&entity, nil, nil)
	if strings.Contains(canonical, "## 产品档案") {
		t.Errorf("canonical layout must not invent sections")
	}
	if !strings.Contains(canonical, "## 来源引用") || !strings.Contains(canonical, "## 别名") {
		t.Errorf("canonical layout lost 来源引用/别名")
	}
}
