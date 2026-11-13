/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package main

import (
	"net/http"
	"path"
	"strings"

	public "infini.sh/coco/.public"
	"infini.sh/coco/config"
	"infini.sh/coco/modules"
	_ "infini.sh/coco/modules"
	_ "infini.sh/coco/plugins"
	"infini.sh/framework"
	api1 "infini.sh/framework/core/api"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/vfs"
	"infini.sh/framework/modules/api"
	"infini.sh/framework/modules/elastic"
	"infini.sh/framework/modules/metrics"
	"infini.sh/framework/modules/pipeline"
	"infini.sh/framework/modules/queue"
	queue2 "infini.sh/framework/modules/queue/disk_queue"
	"infini.sh/framework/modules/security"
	stats2 "infini.sh/framework/modules/stats"
	"infini.sh/framework/modules/task"
	"infini.sh/framework/modules/web"
	_ "infini.sh/framework/plugins"
	stats "infini.sh/framework/plugins/stats_statsd"
)

func main() {

	terminalHeader := ("   ___  ___  ___  ___     _     _____ \n")
	terminalHeader += ("  / __\\/___\\/ __\\/___\\   /_\\    \\_   \\\n")
	terminalHeader += (" / /  //  // /  //  //  //_\\\\    / /\\/\n")
	terminalHeader += ("/ /__/ \\_// /__/ \\_//  /  _  \\/\\/ /_  \n")
	terminalHeader += ("\\____|___/\\____|___/   \\_/ \\_/\\____/  \n\n")
	terminalHeader += ("HOME: https://coco.rs/\n\n")

	terminalFooter := ("")

	app := framework.NewApp("coco", "Coco AI - search, connect, collaborate – all in one place, open-sourced under the GNU AGPLv3.",
		config.Version, config.BuildNumber, config.LastCommitLog, config.BuildDate, config.EOLDate, terminalHeader, terminalFooter)

	app.IgnoreMainConfigMissing()
	app.Init(nil)

	vfs.RegisterFS(public.StaticFS{StaticFolder: global.Env().SystemConfig.WebAppConfig.UI.LocalPath,
		TrimLeftPath:    global.Env().SystemConfig.WebAppConfig.UI.LocalPath,
		CheckLocalFirst: global.Env().SystemConfig.WebAppConfig.UI.LocalEnabled,
		SkipVFS:         !global.Env().SystemConfig.WebAppConfig.UI.VFSEnabled})

	api1.HandleUI("/", spaNoCache(vfs.FileServer(vfs.VFS())))

	defer app.Shutdown()

	if app.Setup(func() {
		module.RegisterSystemModule(&web.WebModule{})
		module.RegisterSystemModule(&security.Module{})
		module.RegisterSystemModule(&api.APIModule{})
		module.RegisterSystemModule(&elastic.ElasticModule{})
		module.RegisterUserPlugin(&stats.StatsDModule{})
		module.RegisterUserPlugin(&task.TaskModule{})
		module.RegisterSystemModule(&queue2.DiskQueue{})
		module.RegisterUserPlugin(&queue.Module{})
		module.RegisterUserPlugin(&pipeline.PipeModule{})
		module.RegisterUserPlugin(&modules.Coco{})
		module.RegisterSystemModule(&metrics.MetricsModule{})
		module.RegisterSystemModule(&stats2.SimpleStatsModule{})

		module.Start()
	}, func() {
	}, nil) {
		app.Run()
	}
}

// spaNoCache keeps the SPA entry point revalidating while hashed assets
// cache freely. The vfs file server only sets Last-Modified, so browsers
// heuristic-cache index.html; after a rebuild the cached entry then
// references asset chunks that no longer exist and the app half-loads
// (missing layouts, failed lazy routes). no-cache forces a cheap
// If-Modified-Since roundtrip on the entry document only.
func spaNoCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean("/" + r.URL.Path)
		if p == "/" || strings.HasSuffix(p, ".html") {
			w.Header().Set("Cache-Control", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}
