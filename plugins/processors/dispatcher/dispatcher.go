/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package dispatcher

import (
	"fmt"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"time"

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

func init() {
	pipeline.RegisterProcessorPlugin(processorName, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{MaxRunningTimeoutInSeconds: 60}
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of flow_replay processor: %s", err)
	}

	runner := Dispatcher{config: &cfg}
	api.HandleUIMethod(api.POST, "/datasource/:id/_reset_sync", runner.reset, api.RequireLogin())
	return &runner, nil
}

func (processor *Dispatcher) Name() string {
	return processorName
}

func (processor *Dispatcher) Process(ctx *pipeline.Context) error {

	// get active datasource list
	ctx1 := orm.NewContextWithParent(ctx)
	ctx1.DirectReadAccess()
	orm.WithModel(ctx1, &common.DataSource{})

	docs := []common.DataSource{}

	from := int64(0)
	size := int64(10)

NextPage:

	if global.ShuttingDown() {
		return nil
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
				return nil
				//return errors.New("shutting down")
			}

			// handle each datasource
			log.Tracef("handle sync task for datasource: %v(%v)", doc.ID, doc.Name)

			if doc.SyncConfig.Enabled {
				// get connector config for the datasource
				lastAccessTime, _ := processor.getLastModifiedTime(doc.ID)
				interval := util.GetDurationOrDefault(doc.SyncConfig.Interval, time.Duration(30*time.Second))

				needSync := false
				if lastAccessTime != "" {
					t := util.ParseTimeWithLocalTZ(lastAccessTime)
					if time.Since(t) > interval {
						needSync = true
						log.Tracef("need to sync, beyond interval, datasource: %v(%v), last_access: %v, interval: %v", doc.ID, doc.Name, lastAccessTime, interval)
					}
				} else {
					needSync = true
					log.Tracef("need to sync, empty interval, datasource: %v(%v), interval: %v", doc.ID, doc.Name, interval)
				}

				//need to sync
				if needSync {

					// handle the sync task for each datasource
					err := processor.syncDatasource(&doc)
					if err != nil {
						log.Errorf("sync error, %v, datasource: %v(%v), last_access: %v, interval: %v", err, doc.ID, doc.Name, lastAccessTime, interval)
						continue
					}

					//update last access time
					err = processor.saveLastModifiedTime(doc.ID, util.FormatTimeWithLocalTZ(time.Now()))
					if err != nil {
						panic(err)
					}
					log.Infof("sync success, update last access time, datasource: %v(%v), last_access: %v, interval: %v", doc.ID, doc.Name, lastAccessTime, interval)
				} else {
					//no need to sync, within interval
					log.Debugf("no need to sync, datasource: %v(%v), last_access: %v, interval: %v", doc.ID, doc.Name, lastAccessTime, interval)
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
