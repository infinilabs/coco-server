/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package integration

import (
	"io"
	"net/http"
	"strings"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

var ver = util.GetUUID()

// aliasSeparator joins the tenant and alias parts in the public widget URL,
// e.g. /integration/infinilabs:coco-website-searchbox/widget. ":" never occurs
// in the server-generated UUID document IDs so the two forms cannot collide,
// and it is excluded from the alias charset, making the split unambiguous.
const aliasSeparator = ":"

func (h *APIHandler) widgetWrapper(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	integrationID := ps.MustGetParameter("id")

	obj, ok := h.resolveIntegration(req, integrationID)
	if !ok {
		h.WriteJSON(w, util.MapStr{
			"_id":    integrationID,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	h.renderWidgetWrapper(w, obj)
}

// Helper function to resolve a widget wrapper request into the target
// integration: a "tenant:alias" value is looked up by the alias pair, any
// other value is treated as a document ID.
func (h *APIHandler) resolveIntegration(req *http.Request, idOrAlias string) (*core.Integration, bool) {
	if strings.Contains(idOrAlias, aliasSeparator) {
		tenant, alias, _ := strings.Cut(idOrAlias, aliasSeparator)
		integrations := []core.Integration{}
		err, _ := orm.SearchWithJSONMapper(&integrations, &orm.Query{
			Size:  1,
			Conds: orm.And(orm.Eq("tenant", tenant), orm.Eq("alias", alias)),
		})
		if err != nil || len(integrations) == 0 {
			return nil, false
		}
		return &integrations[0], true
	}

	obj := core.Integration{}
	obj.ID = idOrAlias
	ctx := orm.NewContextWithParent(req.Context()).DirectReadAccess()
	ctx.PermissionScope(security.PermissionScopePublic)
	exists, err := orm.GetV2(ctx, &obj)
	if !exists || err != nil {
		return nil, false
	}
	return &obj, true
}

// Helper function to render the widget wrapper JS for a resolved integration,
// keyed by its real document ID so downstream APP-INTEGRATION-ID header auth
// keeps working regardless of whether the wrapper was resolved by ID or alias.
func (h *APIHandler) renderWidgetWrapper(w http.ResponseWriter, obj *core.Integration) {
	if !obj.Enabled {
		h.WriteJavascriptHeader(w)
		h.WriteHeader(w, 200)
		return
	}

	var str string

	switch obj.Type {
	//'embedded', 'floating', 'all', 'fullscreen', 'page', 'modal'
	case "fullscreen", "page", "modal":
		if h.fullscreenWrapperTemplate == nil {
			panic("invalid wrapper template")
		}

		info := common.AppConfig()
		str = h.fullscreenWrapperTemplate.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
			switch tag {
			case "ID":
				return w.Write([]byte(obj.ID))
			case "VER":
				return w.Write([]byte(ver))
			case "ENDPOINT":
				endpoint := strings.TrimRight(info.ServerInfo.Endpoint, "/")
				return w.Write([]byte(endpoint))
			}
			return -1, nil
		})
	default:

		if h.searchBoxWrapperTemplate == nil {
			panic("invalid wrapper template")
		}

		info := common.AppConfig()
		str = h.searchBoxWrapperTemplate.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
			switch tag {
			case "ID":
				return w.Write([]byte(obj.ID))
			case "VER":
				return w.Write([]byte(ver))
			case "ENDPOINT":
				endpoint := strings.TrimRight(info.ServerInfo.Endpoint, "/")
				return w.Write([]byte(endpoint))
			}
			return -1, nil
		})
	}
	h.WriteJavascriptHeader(w)
	_, _ = h.Write(w, []byte(str))
	h.WriteHeader(w, 200)

}
