/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package dispatcher

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"infini.sh/coco/core"
	connectorcommon "infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/task/chrono"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/pipeline"
)

type Dispatcher struct {
	api.Handler
	Queue *queue.QueueConfig `config:"queue"`

	config *Config
}

const processorName = "connector_dispatcher"

// activeDispatcher is the live processor instance, set by New(). Routes
// below delegate to it at request time.
var activeDispatcher *Dispatcher

func init() {
	pipeline.RegisterProcessorPluginWithConfigMetadata(processorName, New, Config{})
	pipeline.RegisterProcessorContract(processorName, pipeline.RecordContract{Scope: pipeline.ScopeBatch})

	// NOTE: routes must be registered at package-init time. StartWeb
	// snapshots registeredUIMethodHandler into the live httprouter at boot,
	// and processor constructors (New) run after that — routes registered
	// inside New() never reach the router (this is why /_reset_sync had
	// silently 404'd since its introduction).
	registerDispatcherRoute(api.POST, "/datasource/:id/_reset_sync", func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		activeDispatcher.resetAccessTime(w, req, ps)
	})
	registerDispatcherRoute(api.POST, "/datasource/:id/_sync", func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		activeDispatcher.triggerSync(w, req, ps)
	})
}

// registerDispatcherRoute wraps a UI-method registration with a readiness
// guard: requests arriving before the dispatcher processor is constructed
// get a clean 503 instead of a nil-pointer panic.
func registerDispatcherRoute(method api.Method, pattern string, invoke func(w http.ResponseWriter, req *http.Request, ps httprouter.Params)) {
	api.HandleUIMethod(method, pattern, func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		if activeDispatcher == nil {
			http.Error(w, "dispatcher is not ready", http.StatusServiceUnavailable)
			return
		}
		invoke(w, req, ps)
	}, api.RequireLogin())
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{MaxRunningTimeoutInSeconds: 60}
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of flow_replay processor: %s", err)
	}

	runner := Dispatcher{config: &cfg}
	activeDispatcher = &runner
	return &runner, nil
}

func (processor *Dispatcher) Name() string {
	return processorName
}

func (processor *Dispatcher) Process(ctx *pipeline.Context) error {

	// get active datasource list
	ctx1 := orm.NewContextWithParent(ctx)
	ctx1.DirectReadAccess()
	ctx1.PermissionScope(security.PermissionScopePlatform)

	orm.WithModel(ctx1, &core.DataSource{})

	docs := []core.DataSource{}

	from := int64(0)
	size := int64(10)

NextPage:

	if global.ShuttingDown() {
		log.Debugf("[%s] shutting down, skipping pagination", processor.Name())
		return errors.New("shutting down")
	}

	builder := orm.NewQuery()
	builder.From(int(from))
	builder.Size(int(size))
	builder.SortBy(orm.Sort{Field: "created", SortType: orm.ASC})
	builder.Filter(orm.TermQuery("sync.enabled", true), orm.TermQuery("enabled", true))

	err, res := elastic.SearchV2WithResultItemMapper(ctx1, &docs, builder, nil)
	if err != nil {
		panic(err)
	}

	if len(docs) > 0 {
		for _, doc := range docs {

			if global.ShuttingDown() {
				log.Debugf("[%s] shutting down, skipping remaining datasources", processor.Name())
				return errors.New("shutting down")
			}

			// handle each datasource
			log.Tracef("handle sync task for datasource: %v(%v)", doc.ID, doc.Name)

			if common.IsDatasourceDeleted(doc.ID) {
				log.Tracef("skip deleted datasource: %v(%v)", doc.ID, doc.Name)
				continue
			}

			if doc.SyncConfig.Enabled {
				// close the sync-run journal when its dispatch pipeline has
				// ended — otherwise a completed fetch reads as stale forever
				if connectorcommon.CloseSyncRunIfIdle(doc.ID) {
					// the run just closed: take the observed-ID set (always,
					// so non-whitelisted connectors don't leak registry
					// memory) and reconcile deletions for full-scan ones —
					// missing docs get soft tombstones (source_deleted), not
					// deletes; humans decide via the document list badge
					seen := connectorcommon.TakeSeenIDs(doc.ID)
					if connectorcommon.ShouldReconcile(&doc) {
						go connectorcommon.ReconcileDeletedDocs(&doc, seen)
					}
				}

				// strategy gate (P2-1): interval and cron sources auto-dispatch.
				// manual sources wait for POST /datasource/:id/_sync;
				// realtime sources are fed by webhook pushes. The journal
				// close above still applies — manual/realtime runs need
				// their journals closed too.
				if strategy := doc.SyncConfig.EffectiveStrategy(); strategy == core.SyncStrategyInterval || strategy == core.SyncStrategyCron {
					// get connector config for the datasource
					lastAccessTime, _ := processor.getLastAccessTime(doc.ID)
					var needSync bool
					if strategy == core.SyncStrategyCron {
						needSync = doc.SyncConfig.CronDue(lastAccessTime, time.Now())
						if !needSync {
							if _, err := chrono.ParseCronExpression(strings.TrimSpace(doc.SyncConfig.Crontab)); err != nil {
								log.Warnf("cron datasource [%s] has invalid crontab %q — no auto dispatch; manual trigger still works", doc.ID, doc.SyncConfig.Crontab)
							}
						}
					} else {
						interval := util.GetDurationOrDefault(doc.SyncConfig.Interval, time.Duration(30*time.Second))

						needSync = false
						if lastAccessTime != "" {
							t := util.ParseTimeWithLocalTZ(lastAccessTime)
							if time.Since(t) > interval {
								needSync = true
								log.Tracef("need to sync, beyond interval, datasource: %v(%v), last_access: %v, interval: %v", doc.ID, doc.Name, lastAccessTime, interval)
							} else {
								log.Tracef("time: %v, %v since interval: %v", t, time.Since(t), interval)
							}
						} else {
							needSync = true
							log.Tracef("need to sync, empty interval, datasource: %v(%v), interval: %v", doc.ID, doc.Name, interval)
						}
					}

					//need to sync
					if needSync {
						log.Debugf("start sync, datasource: %v(%v), last_access: %v, strategy: %v", doc.ID, doc.Name, lastAccessTime, strategy)
						// handle the sync task for each datasource
						err := processor.syncDatasource(&doc)
						if err != nil {
							log.Errorf("sync error, %v, datasource: %v(%v), last_access: %v, strategy: %v", err, doc.ID, doc.Name, lastAccessTime, strategy)
							continue
						}

						//update last access time
						err = processor.saveLastAccessTime(doc.ID, util.FormatTimeWithLocalTZ(time.Now()))
						if err != nil {
							panic(err)
						}
						log.Debugf("sync success, update last access time, datasource: %v(%v), last_access: %v, strategy: %v", doc.ID, doc.Name, lastAccessTime, strategy)
					} else {
						//no need to sync, within schedule
						log.Debugf("no need to sync, datasource: %v(%v), last_access: %v, strategy: %v", doc.ID, doc.Name, lastAccessTime, strategy)
					}
				}
			}
		}
	}

	if res.Total > (from + size) {
		from = from + size
		docs = docs[:0]
		goto NextPage
	}

	return nil
}
