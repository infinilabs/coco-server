/* Copyright © INFINI LTD.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import "infini.sh/framework/core/orm"

// PipelineDeployment is the desired state of a pipeline rollout across
// gateway/agent instances: pick a pipeline (coco_pipelineconfigv2 row) and
// targets (explicit instance ids or a group selector); the reconciler in
// modules/pipeline converges the fleet by delivering the pipeline config
// file through the framework configs channel (managed-configs), where
// instances pick it up on their next config sync (fsnotify reload).
type DeploymentPlacement struct {
	// Selector holds reserved-key targets; key "group" is a comma-separated
	// group list (any-match, re-evaluated every reconciler tick).
	Selector map[string]string `json:"selector,omitempty" elastic_mapping:"selector:{type:object,enabled:false}"`
	// Instances are explicit target instance ids (used when no selector).
	Instances []string `json:"instances,omitempty" elastic_mapping:"instances:{type:keyword}"`
	// Strategy: "all" deploys to every resolved target; "singleton" to the
	// first one only.
	Strategy string `json:"strategy" elastic_mapping:"strategy:{type:keyword}"`
}

type PipelineDeployment struct {
	orm.ORMObjectBase
	Name       string              `json:"name" elastic_mapping:"name:{type:keyword}"`
	PipelineID string              `json:"pipeline_id" elastic_mapping:"pipeline_id:{type:keyword}"`
	Enabled    bool                `json:"enabled" elastic_mapping:"enabled:{type:boolean}"`
	Version    int64               `json:"version" elastic_mapping:"version:{type:long}"`
	Placement  DeploymentPlacement `json:"placement" elastic_mapping:"placement:{type:object}"`
}
