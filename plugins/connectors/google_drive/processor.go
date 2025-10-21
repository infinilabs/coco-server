/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package google_drive

import (
	"fmt"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"time"
)

const NAME = "google_drive"

type Processor struct {
	cmn.ConnectorProcessorBase
	SkipInvalidToken bool   `json:"skip_invalid_token" config:"skip_invalid_token"`
	Timeout          string `json:"timeout" config:"timeout"`
	timeout          time.Duration
}

func init() {
	pipeline.RegisterProcessorPlugin(NAME, New)
	api.HandleUIMethod(api.GET, "/connector/:id/oauth_connect", connect, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/connector/:id/oauth_redirect", oAuthRedirect, api.RequireLogin())
}

func New(c *config.Config) (pipeline.Processor, error) {
	runner := Processor{SkipInvalidToken: true}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of processor %v, error: %s", NAME, err)
	}

	runner.timeout = util.GetDurationOrDefault(runner.Timeout, 30*time.Second)

	runner.Init(c, &runner)
	return &runner, nil
}

func (processor *Processor) Name() string {
	return NAME
}
