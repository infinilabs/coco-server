/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package dispatcher

import (
	"net/http"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

// triggerSync handles POST /datasource/:id/_sync (P2-2): dispatch one sync
// run for the datasource immediately, regardless of its interval schedule.
// This is the only dispatch entry for manual-strategy sources and the
// UI's "sync now" button; interval sources may also use it to pull a run
// forward. The singleton dispatch pipeline prevents overlapping runs.
func (processor *Dispatcher) triggerSync(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")
	if id == "" {
		panic("invalid id")
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.DirectReadAccess()
	ctx.PermissionScope(security.PermissionScopePlatform)

	doc := core.DataSource{}
	doc.ID = id
	exists, err := orm.GetV2(ctx, &doc)
	if err != nil {
		panic(err)
	}
	if !exists {
		h := util.MapStr{"_id": id, "found": false}
		processor.WriteJSON(w, h, http.StatusNotFound)
		return
	}

	if !doc.Enabled {
		processor.WriteError(w, "datasource is disabled", http.StatusBadRequest)
		return
	}
	if !doc.SyncConfig.Enabled {
		processor.WriteError(w, "datasource sync is disabled", http.StatusBadRequest)
		return
	}
	if doc.Type != "connector" || doc.Connector.ConnectorID == "" {
		processor.WriteError(w, "datasource has no connector to dispatch", http.StatusBadRequest)
		return
	}

	if err := processor.syncDatasource(&doc); err != nil {
		processor.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// stamp last access so a subsequent interval schedule counts from now
	if err := processor.saveLastAccessTime(doc.ID, util.FormatTimeWithLocalTZ(time.Now())); err != nil {
		// non-fatal: the run was dispatched successfully
		log.Warnf("failed to save last access time for datasource [%s]: %v", doc.ID, err)
	}

	processor.WriteOKJSON(w, util.MapStr{
		"_id":       doc.ID,
		"result":    "ok",
		"triggered": true,
	})
}
