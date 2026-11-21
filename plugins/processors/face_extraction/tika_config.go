/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package face_extraction

import (
	"strings"

	"infini.sh/coco/modules/common"
)

// resolveTikaEndpoint picks the Tika server for a pipeline stage: an endpoint
// pinned in the pipeline config wins; otherwise the operator's global
// document-processing setting applies, then the historic default. Resolved per
// call so a settings change lands without rebuilding the pipeline.
func resolveTikaEndpoint(pipelineValue string) string {
	if endpoint := strings.TrimSpace(pipelineValue); endpoint != "" {
		return endpoint
	}
	return common.AppConfig().DocumentProcessing.EffectiveTikaEndpoint()
}

// resolveTikaTimeout picks the Tika call timeout the same way.
func resolveTikaTimeout(pipelineValue int) int {
	if pipelineValue > 0 {
		return pipelineValue
	}
	return common.AppConfig().DocumentProcessing.EffectiveTikaTimeoutInSeconds()
}
