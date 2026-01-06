package attachment

import (
	"infini.sh/coco/core"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
)

func getAttachmentStats(ids []string) map[string]util.MapStr {
	out := make(map[string]util.MapStr)
	for _, id := range ids {
		v, err := kv.GetValue(core.AttachmentStatsBucket, []byte(id))
		if err == nil {
			obj := util.MapStr{}
			util.MustFromJSONBytes(v, &obj)
			out[id] = obj
		} else {
			//TODO remove this when the actual pipeline is ready
			obj := util.MapStr{
				"initial_parsing": "completed",
			}
			out[id] = obj
		}
	}
	return out
}
