package pipeline

import (
	"encoding/json"
	"fmt"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"infini.sh/coco/core/codecs"
	"infini.sh/framework/core/util"
)

var hj = jsoniter.ConfigCompatibleWithStandardLibrary

func TestAllocProbe(t *testing.T) {
	payload := benchDocPayload(1 << 20)
	c := codecs.DocumentCodec{}

	// decode: full codec (raw-map pass + per-field) vs plain stdlib/jsoniter into MapStr
	nCodec := testing.AllocsPerRun(20, func() { c.Decode(payload) })
	nStd := testing.AllocsPerRun(20, func() {
		var m util.MapStr
		json.Unmarshal(payload, &m)
	})
	nIt := testing.AllocsPerRun(20, func() {
		var m util.MapStr
		hj.Unmarshal(payload, &m)
	})
	fmt.Printf("decode allocs codec(cold)=%.0f stdlib-full=%.0f jsoniter-full=%.0f\n", nCodec, nStd, nIt)

	rec, _ := c.Decode(payload)
	doc := rec.Fields["document"].(util.MapStr)
	nEncStd := testing.AllocsPerRun(20, func() { json.Marshal(doc) })
	nEncIt := testing.AllocsPerRun(20, func() { hj.Marshal(doc) })
	fmt.Printf("encode allocs stdlib=%.0f jsoniter=%.0f\n", nEncStd, nEncIt)
}
