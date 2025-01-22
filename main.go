/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package main

import (
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
	"infini.sh/framework/modules/pipeline"
	"infini.sh/framework/modules/queue"
	queue2 "infini.sh/framework/modules/queue/disk_queue"
	"infini.sh/framework/modules/task"
	"infini.sh/framework/modules/web"
	_ "infini.sh/framework/plugins/badger"
	_ "infini.sh/framework/plugins/elastic/bulk_indexing"
	_ "infini.sh/framework/plugins/elastic/indexing_merge"
	_ "infini.sh/framework/plugins/http"
	_ "infini.sh/framework/plugins/queue/consumer"
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

	api1.HandleUI("/", vfs.FileServer(vfs.VFS()))

	defer app.Shutdown()

	if app.Setup(func() {
		module.RegisterSystemModule(&web.WebModule{})
		module.RegisterSystemModule(&api.APIModule{})
		module.RegisterSystemModule(&elastic.ElasticModule{})
		module.RegisterUserPlugin(&stats.StatsDModule{})
		module.RegisterUserPlugin(&task.TaskModule{})
		module.RegisterSystemModule(&queue2.DiskQueue{})
		module.RegisterUserPlugin(&queue.Module{})
		module.RegisterUserPlugin(&pipeline.PipeModule{})
		module.RegisterUserPlugin(&modules.Coco{})

		module.Start()
	}, func() {
	}, nil) {
		app.Run()
	}
}
