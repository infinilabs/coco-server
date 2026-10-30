package wiki

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"
)

var pageImagesStoreOnce sync.Once

func pageImagesSetup(t *testing.T) {
	t.Helper()
	pageImagesStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-page-images.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.Document{}, "document-page-images"},
			{core.Attachment{}, "attachment-page-images"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-page-images-test", handler)
	})
}

func TestStripImageMarkers(t *testing.T) {
	in := "正文第一段。\n[[Image(abc123\t水印噪声 OCR 文本 20231101)]]\n正文第二段。"
	out := StripImageMarkers(in)
	assert.Contains(t, out, "正文第一段。")
	assert.Contains(t, out, "正文第二段。")
	assert.NotContains(t, out, "[[Image")
	assert.NotContains(t, out, "水印噪声", "OCR text must not leak into compiled corpus")
	assert.Equal(t, "纯正文", StripImageMarkers("纯正文"), "no-marker content passes through untouched")
	assert.Equal(t, "", StripImageMarkers("[[Image(only\t扫描件)]]"), "a scan-only doc loses its OCR as compilation input")
}

func TestFigureOverlaps(t *testing.T) {
	heading, title := "部署架构", "网关部署指南"
	assert.True(t, figureOverlaps("部署架构示意图：网关与队列拓扑", heading, title), "CJK bigrams overlap")
	assert.True(t, figureOverlaps("gateway topology overview", heading, "Gateway Deployment"), "latin token match")
	assert.False(t, figureOverlaps("发票抬头 金额 税号", heading, title), "unrelated OCR never places")
	assert.False(t, figureOverlaps("", heading, title), "no OCR text = no relevance proof")
	assert.False(t, figureOverlaps("20231101 版本号", "", ""), "no heading/title = no gate")
}

func TestImageAlt(t *testing.T) {
	assert.Equal(t, "部署架构 图 含换行", imageAlt("部署架构\t图\n含换行", "fallback"))
	assert.Equal(t, "fallback", imageAlt("  ", "fallback"))
	long := strings.Repeat("长", 100)
	assert.LessOrEqual(t, len([]rune(imageAlt(long, "f"))), 61, "alt truncated with ellipsis")
	assert.NotContains(t, imageAlt("[Figure](x)", "f"), "[", "markdown-breaking chars stripped")
}

func TestCollectSectionImages(t *testing.T) {
	pageImagesSetup(t)
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh

	relevant := core.Attachment{}
	relevant.SetID("att-relevant")
	relevant.Name = "arch-diagram.png"
	relevant.MimeType = "image/png"
	relevant.URL = "/attachment/att-relevant"
	relevant.Text = "部署架构示意图：网关与消息队列拓扑"
	require.NoError(t, orm.Create(ctx, &relevant))

	irrelevant := core.Attachment{}
	irrelevant.SetID("att-irrelevant")
	irrelevant.Name = "invoice.png"
	irrelevant.MimeType = "image/png"
	irrelevant.URL = "/attachment/att-irrelevant"
	irrelevant.Text = "发票抬头与税号"
	require.NoError(t, orm.Create(ctx, &irrelevant))

	doc1 := core.Document{}
	doc1.SetID("pi-doc1")
	doc1.Title = "部署手册"
	doc1.Attachments = []string{"att-relevant", "att-irrelevant"}
	require.NoError(t, orm.Create(ctx, &doc1))

	doc2 := core.Document{}
	doc2.SetID("pi-doc2")
	doc2.Title = "财务制度"
	doc2.Attachments = []string{"att-relevant"}
	require.NoError(t, orm.Create(ctx, &doc2))

	sections := []core.WikiSection{
		{ID: "s1", Heading: "部署架构", Body: "网关与队列的部署拓扑说明。", DocIDs: []string{"pi-doc1"}},
		{ID: "s2", Heading: "报销流程", Body: "报销的单据要求。", DocIDs: []string{"pi-doc2"}},
	}
	titles := map[string]string{"pi-doc1": "部署手册", "pi-doc2": "财务制度"}
	images := collectSectionImages(ctx, "网关部署指南", sections, titles)
	require.Len(t, images, 1, "only the section-overlapping figure places")
	assert.Equal(t, "s1", images[0].SectionID)
	assert.Equal(t, "/attachment/att-relevant", images[0].URL)
	assert.Equal(t, "pi-doc1", images[0].DocID)

	md := RenderTopicContentWithImages(sections, nil, titles, images)
	assert.Contains(t, md, "![部署架构示意图", "figure renders inline after its section")
	assert.Contains(t, md, "(/attachment/att-relevant)")
	assert.Contains(t, md, "[部署手册](#/preview/document/pi-doc1)", "figure cites its source document")

	// no citations → no queries, no images
	assert.Empty(t, collectSectionImages(ctx, "t", []core.WikiSection{{ID: "s", Heading: "h"}}, titles))
}
