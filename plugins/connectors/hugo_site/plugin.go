/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package hugo_site

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"net/url"
	"strings"
	"time"
)

type Plugin struct {
	api.Handler

	Enabled          bool               `config:"enabled"`
	Interval         string             `config:"interval"`
	SkipInvalidToken bool               `config:"skip_invalid_token"`
	Urls             []string           `config:"urls"`
	Queue            *queue.QueueConfig `config:"queue"`
}

func (this *Plugin) Setup() {
	ok, err := env.ParseConfig("connector.hugo_site", &this)
	if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}

	if !this.Enabled {
		return
	}

	if this.Queue == nil {
		this.Queue = &queue.QueueConfig{Name: "indexing_documents"}
	}

	//api.HandleAPIMethod(api.GET, "/connector/google_drive/connect", this.connect)
	//api.HandleAPIMethod(api.POST, "/connector/google_drive/reset", this.reset)
	//api.HandleAPIMethod(api.GET, "/connector/google_drive/oauth_redirect", this.oAuthRedirect)

}

func (this *Plugin) Start() error {

	if this.Enabled {
		task.RegisterScheduleTask(task.ScheduleTask{
			ID:          util.GetUUID(),
			Group:       "connectors",
			Singleton:   true,
			Interval:    util.GetDurationOrDefault(this.Interval, time.Second*30).String(),
			Description: "indexing hugo json docs",
			Task: func(ctx context.Context) {
				for _, url := range this.Urls {
					log.Infof("fetch hugo url: %v", url)

					res,err:=util.HttpGet(url)
					if err!=nil{
						panic(err)
					}

					if res.Body!=nil{
						var documents []HugoDocument

						// Unmarshal JSON into the slice
						err := util.FromJSONBytes(res.Body,&documents)
						if err != nil {
							panic(errors.Errorf("Failed to parse JSON: %v", err))
						}

						// Output the parsed data
						for i, v := range documents {
							doc:=common.Document{Source: common.DataSourceReference{Type: "connector",Name: "hugo_site"}}
							doc.Type="web_page"
							doc.Icon="web"
							doc.Title=v.Title
							doc.Content=v.Content
							doc.Category=v.Category
							doc.Subcategory=v.Subcategory
							doc.Summary=v.Summary
							doc.Tags=v.Tags
							v2,er:=getFullURL(url,v.URL)
							if er!=nil{
								panic(er)
							}
							doc.URL=v2
							log.Infof("Document %d: %+v %v", i+1, doc.Title, doc.URL)
							doc.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", "test", "hugo-site", doc.URL))

							data := util.MustToJSONBytes(doc)

							if global.Env().IsDebug {
								log.Tracef(string(data))
							}

							err := queue.Push(queue.SmartGetOrInitConfig(this.Queue), data)
							if err != nil {
								panic(err)
							}
						}

					}

				}

			},
		})

	}

	return nil
}

// Function to construct the full URL using only the domain from the seed URL
func getFullURL(seedURL, relativePath string) (string, error) {
	// Parse the seed URL
	parsedURL, err := url.Parse(seedURL)
	if err != nil {
		return "", fmt.Errorf("invalid seed URL: %w", err)
	}

	// Extract the domain (scheme and host)
	domain := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// Remove any leading "/" from relativePath to avoid duplication
	relativePath = strings.TrimPrefix(relativePath, "/")

	// Combine the domain with the relative path
	fullURL := fmt.Sprintf("%s/%s", domain, relativePath)

	return fullURL, nil
}

func (this *Plugin) Stop() error {
	return nil
}

func (this *Plugin) Name() string {
	return "hugo_site"
}

func init() {
	module.RegisterUserPlugin(&Plugin{SkipInvalidToken: true})
}

