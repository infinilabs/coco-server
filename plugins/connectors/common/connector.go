/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/pipeline"
)

type ConnectorAPI interface {
	Fetch(pipeCtx *pipeline.Context, connector *common.Connector, datasource *common.DataSource) error
}
