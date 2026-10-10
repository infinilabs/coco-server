/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/entity_card"
	"infini.sh/framework/core/orm"
)

// Wiki entities on the standard entity-card contract (W10): the
// framework owns /entity/card/:type/:id and /entity/label/_batch_get and
// routes them to registered EntityProviders (the enterprise plugin
// registers user/team the same way). Wiki entities register under the
// "wiki_entity" type — every EntityCard/EntityLabel site in the console
// (search chips, wikilink hovers, chat references) renders them with
// zero frontend change.

// CardTypeWikiEntity is the provider type name.
const CardTypeWikiEntity = "wiki_entity"

type WikiEntityProvider struct{}

func init() {
	entity_card.RegisterEntityProvider(CardTypeWikiEntity, &WikiEntityProvider{})
}

// GenEntityInfo builds the card for one entity. A missing entity yields
// an id-titled shell — the card surface degrades gracefully, never 404s.
func (p *WikiEntityProvider) GenEntityInfo(ctx context.Context, t, id string) *entity_card.EntityInfo {
	entity, ok := findEntityForCard(ctx, id)
	if !ok {
		return &entity_card.EntityInfo{Type: t, ID: id, Title: id}
	}

	card := &entity_card.EntityInfo{
		Type:     t,
		ID:       entity.ID,
		Color:    "#1677ff",
		Icon:     "font_book",
		Title:    entity.Name,
		Subtitle: entity.Type,
	}
	if len(entity.Aliases) > 0 {
		card.Tags = entity.Aliases
	}
	if entity.ArticleID != "" {
		card.URL = articleURLForCard(entity.ArticleID)
	}
	card.Properties = append(card.Properties, entity_card.Property{
		Value: "status: " + entity.Status,
	})
	if len(entity.Properties) > 0 {
		rows := make([]entity_card.CardRow, 0, len(entity.Properties))
		for k, v := range entity.Properties {
			rows = append(rows, entity_card.CardRow{
				Columns: []entity_card.CardColumn{
					{Value: k},
					{Value: v},
				},
			})
		}
		card.Details = &entity_card.CardDetails{Table: &entity_card.CardTable{Rows: rows}}
	}
	return card
}

// GenEntityLabel resolves ids to {id, title, icon} rows; unknown ids
// degrade to id-titled rows so batch consumers never see gaps.
func (p *WikiEntityProvider) GenEntityLabel(ctx context.Context, t string, ids []string) []entity_card.EntityLabel {
	out := make([]entity_card.EntityLabel, 0, len(ids))
	if len(ids) == 0 {
		return out
	}

	found := map[string]*core.WikiEntity{}
	octx := orm.NewContext()
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiEntity{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(len(ids)).
		Filter(orm.TermsQuery("id", ids)).
		Include("id", "name", "type"))
	if err == nil {
		entities, _, derr := elastic.DecodeHits[core.WikiEntity](res)
		if derr == nil {
			for i := range entities {
				found[entities[i].ID] = &entities[i]
			}
		}
	}

	for _, id := range ids {
		label := entity_card.EntityLabel{Type: t, ID: id, Icon: "font_book", Title: id}
		if e, ok := found[id]; ok {
			label.Title = e.Name
			label.Subtitle = e.Type
		}
		out = append(out, label)
	}
	return out
}

func articleURLForCard(articleID string) string {
	endpoint := ""
	if cfg := companyMapConfigFn(); cfg.ServerInfo != nil {
		endpoint = cfg.ServerInfo.Endpoint
	}
	return endpoint + "/#/wiki/article/" + articleID
}

func findEntityForCard(ctx context.Context, id string) (*core.WikiEntity, bool) {
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiEntity{})
	entity := core.WikiEntity{}
	entity.ID = id
	exists, err := orm.GetV2(octx, &entity)
	if err != nil || !exists {
		return nil, false
	}
	return &entity, true
}
