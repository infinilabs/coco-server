/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package main

import (
	"infini.sh/coco/config"
	_ "infini.sh/coco/modules"
	"infini.sh/framework"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/util"
	apiModule "infini.sh/framework/modules/api"
	stats "infini.sh/framework/plugins/stats_statsd"
)

func main() {

	terminalHeader := ("   ___  ___  ___  ___     _     _____ \n")
	terminalHeader += ("  / __\\/___\\/ __\\/___\\   /_\\    \\_   \\\n")
	terminalHeader += (" / /  //  // /  //  //  //_\\\\    / /\\/\n")
	terminalHeader += ("/ /__/ \\_// /__/ \\_//  /  _  \\/\\/ /_  \n")
	terminalHeader += ("\\____|___/\\____|___/   \\_/ \\_/\\____/  \n\n")

	terminalFooter := ("")

	app := framework.NewApp("coco", "Coco AI - search, connect, collaborate – all in one place.",
		util.TrimSpaces(config.Version), util.TrimSpaces(config.BuildNumber), util.TrimSpaces(config.LastCommitLog), util.TrimSpaces(config.BuildDate), util.TrimSpaces(config.EOLDate), terminalHeader, terminalFooter)

	app.IgnoreMainConfigMissing()
	app.Init(nil)

	defer app.Shutdown()

	if app.Setup(func() {
		module.RegisterSystemModule(&apiModule.APIModule{})
		module.RegisterUserPlugin(&stats.StatsDModule{})

		//vfs.RegisterFS(ui.StaticFS{StaticFolder: global.Env().SystemConfig.WebAppConfig.UI.LocalPath,
		//	TrimLeftPath:    global.Env().SystemConfig.WebAppConfig.UI.LocalPath,
		//	CheckLocalFirst: global.Env().SystemConfig.WebAppConfig.UI.LocalEnabled,
		//	SkipVFS:         !global.Env().SystemConfig.WebAppConfig.UI.VFSEnabled})

		module.Start()
	}, func() {
	}, nil) {
		app.Run()
	}
}
