/* Copyright © INFINI LTD.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package pipeline

// The unified pipeline studio endpoints: POST /pipeline/dry-run replays a
// processor chain over sample documents and reports each step's field
// effects; POST /pipeline/ai-generate dispatches to the registered chain
// generator (see ai_generate.go); GET /pipeline/codecs lists the record
// codecs samples can be decoded with.
//
// These routes belong to the upstream framework's pipeline module; the
// local framework copy ships only the Go-side registries (chain generator,
// record codecs) without the HTTP surface, so coco registers the endpoints
// here until upstream pushes the real implementation.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"time"

	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/framework/core/api"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/event"
	fpipeline "infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	pipemod "infini.sh/framework/modules/pipeline"
)

const (
	maxDryRunSamples     = 10
	defaultRecordCodec   = "coco_document"
	dryRunSideEffectsCap = 100
)

type StudioAPIHandler struct {
	api.Handler
}

func init() {
	h := StudioAPIHandler{}
	api.HandleUIMethod(api.POST, "/pipeline/dry-run", h.dryRun, api.RequireLogin(), api.RequirePermission(security.PermissionSystemPipelineRead))
	api.HandleUIMethod(api.POST, "/pipeline/ai-generate", h.aiGenerate, api.RequireLogin(), api.RequirePermission(security.PermissionSystemPipelineRead))
	api.HandleUIMethod(api.GET, "/pipeline/codecs", h.listCodecs, api.RequireLogin(), api.RequirePermission(security.PermissionSystemPipelineRead))
}

// ─── dry-run ────────────────────────────────────────────────────────────────

type dryRunSampleIn struct {
	Raw   string `json:"raw"`
	Codec string `json:"codec"`
}

type dryRunOptionsIn struct {
	ScopeCheck  bool   `json:"scope_check"`
	SideEffects string `json:"side_effects"`
	LLMPolicy   string `json:"llm_policy"`
}

type dryRunRequestIn struct {
	Processor []map[string]interface{} `json:"processor"`
	Samples   []dryRunSampleIn         `json:"samples"`
	Options   dryRunOptionsIn          `json:"options"`
}

// report shapes mirror the shared @infinilabs/pipeline-ui models
// (pipeline-api.ts DryRunReport/StepResult/StepDesc) — do not rename keys.

type stepDesc struct {
	Step      int    `json:"step"`
	Processor string `json:"processor"`
	OK        bool   `json:"ok"`
	Note      string `json:"note,omitempty"`
}

type fieldDiff struct {
	Added   map[string]interface{} `json:"added,omitempty"`
	Removed map[string]interface{} `json:"removed,omitempty"`
	Changed map[string]interface{} `json:"changed,omitempty"`
}

type stepResult struct {
	Step       int                    `json:"step"`
	Processor  string                 `json:"processor"`
	Before     map[string]interface{} `json:"before,omitempty"`
	After      map[string]interface{} `json:"after,omitempty"`
	Diff       fieldDiff              `json:"diff"`
	DurationMS int64                  `json:"duration_ms"`
	Outcome    string                 `json:"outcome"`
	Err        string                 `json:"err,omitempty"`
}

type dryRunReport struct {
	Compiled    []stepDesc               `json:"compiled"`
	Steps       [][]stepResult           `json:"steps"`
	SideEffects []fpipeline.DryRunWrite  `json:"side_effects"`
}

func entryName(entry map[string]interface{}) string {
	if _, ok := entry["if"]; ok {
		return "if"
	}
	for k := range entry {
		return k
	}
	return "?"
}

// snapshotFields deep-copies the record payload through JSON so per-step
// before/after states are independent of later mutation.
func snapshotFields(in util.MapStr) map[string]interface{} {
	b, err := json.Marshal(in)
	if err != nil {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}

func diffFields(before, after map[string]interface{}) fieldDiff {
	diff := fieldDiff{}
	if before == nil {
		before = map[string]interface{}{}
	}
	if after == nil {
		after = map[string]interface{}{}
	}
	for k, v := range after {
		prev, existed := before[k]
		if !existed {
			if diff.Added == nil {
				diff.Added = map[string]interface{}{}
			}
			diff.Added[k] = v
		} else if !reflect.DeepEqual(prev, v) {
			if diff.Changed == nil {
				diff.Changed = map[string]interface{}{}
			}
			diff.Changed[k] = v
		}
	}
	for k, v := range before {
		if _, existed := after[k]; !existed {
			if diff.Removed == nil {
				diff.Removed = map[string]interface{}{}
			}
			diff.Removed[k] = v
		}
	}
	return diff
}

// safeProcess guards against processor panics so one broken processor
// returns a report instead of taking down the request.
func safeProcess(p fpipeline.Processor, ctx *fpipeline.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("processor panic: %v", r)
		}
	}()
	return p.Process(ctx)
}

func (h *StudioAPIHandler) dryRun(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body := dryRunRequestIn{}
	if req.ContentLength != 0 {
		if err := h.DecodeJSON(req, &body); err != nil {
			h.Error400(w, err.Error())
			return
		}
	}
	if len(body.Samples) == 0 {
		h.Error400(w, "no samples: provide at least one document under samples[]")
		return
	}
	if len(body.Samples) > maxDryRunSamples {
		body.Samples = body.Samples[:maxDryRunSamples]
	}

	// compile phase: one entry at a time so a failing entry degrades to a
	// per-entry compile note (compiled[i].ok=false) instead of a 500
	compiled := make([]stepDesc, 0, len(body.Processor))
	var procs []fpipeline.Processor
	for i, entry := range body.Processor {
		cfg, err := config.NewConfigFrom(entry)
		if err == nil {
			var compiled1 *fpipeline.Processors
			compiled1, err = fpipeline.NewPipeline([]*config.Config{cfg})
			if err == nil {
				procs = append(procs, compiled1.List...)
			}
		}
		note := ""
		if err != nil {
			note = err.Error()
		}
		compiled = append(compiled, stepDesc{Step: i, Processor: entryName(entry), OK: err == nil, Note: note})
		if err != nil {
			break // later steps can't compile past a broken entry
		}
	}

	captureEffects := body.Options.SideEffects != "passthrough"
	steps := make([][]stepResult, 0, len(body.Samples))
	sideEffects := make([]fpipeline.DryRunWrite, 0)

	for _, s := range body.Samples {
		rec, decodeErr := decodeSample(s)
		if decodeErr != nil {
			steps = append(steps, []stepResult{{Outcome: "error", Err: decodeErr.Error()}})
			continue
		}

		ctx := fpipeline.AcquireContext(fpipeline.PipelineConfigV2{})
		ctx.Starting()
		var sink []fpipeline.DryRunWrite
		if captureEffects {
			fpipeline.EnableDryRun(ctx, &sink, true)
		}

		sampleSteps := make([]stepResult, 0, len(procs))
		for j, proc := range procs {
			before := snapshotFields(rec.Fields)
			ctx.Set(fpipeline.RecordContextKey, rec)
			ctx.Set(fpipeline.CloneContextKey, &[]*event.Event{})
			t0 := time.Now()
			perr := safeProcess(proc, ctx)
			took := time.Since(t0).Milliseconds()
			after := snapshotFields(rec.Fields)
			res := stepResult{
				Step:       j,
				Processor:  proc.Name(),
				Before:     before,
				After:      after,
				Diff:       diffFields(before, after),
				DurationMS: took,
				Outcome:    "ok",
			}
			if perr != nil {
				res.Outcome = "error"
				res.Err = perr.Error()
				sampleSteps = append(sampleSteps, res)
				break // one error skips the rest of the chain for this record
			}
			if fpipeline.IsDropped(rec) {
				res.Outcome = "dropped"
				sampleSteps = append(sampleSteps, res)
				break
			}
			sampleSteps = append(sampleSteps, res)
		}
		steps = append(steps, sampleSteps)

		if len(sink) > dryRunSideEffectsCap {
			sink = sink[:dryRunSideEffectsCap]
		}
		sideEffects = append(sideEffects, sink...)

		fpipeline.ReleaseContext(ctx)
	}

	h.WriteJSON(w, dryRunReport{Compiled: compiled, Steps: steps, SideEffects: sideEffects}, http.StatusOK)
}

// lookupRecordCodec resolves a codec by name WITHOUT the default-codec
// fallback GetRecordCodec applies — an unknown name must fall through to
// the plain-JSON decode path, not silently decode as the default codec.
// Local shim over the framework registry until upstream ships the helper.
func lookupRecordCodec(name string) (fpipeline.RecordCodec, bool) {
	for _, c := range fpipeline.ListRecordCodecs() {
		if c.Name == name {
			return fpipeline.GetRecordCodec(name), true
		}
	}
	return nil, false
}

// recordCodecNames lists registered codec names for error messages and the
// codec picker. Local shim over the framework registry.
func recordCodecNames() []string {
	infos := fpipeline.ListRecordCodecs()
	names := make([]string, 0, len(infos))
	for _, c := range infos {
		names = append(names, c.Name)
	}
	sort.Strings(names)
	return names
}

func decodeSample(s dryRunSampleIn) (*event.Event, error) {
	codecName := s.Codec
	if codecName == "" {
		codecName = defaultRecordCodec
	}
	if codec, ok := lookupRecordCodec(codecName); ok {
		return codec.Decode([]byte(s.Raw))
	}
	// unknown codec: fall back to a plain JSON object payload
	rec := &event.Event{Fields: util.MapStr{}}
	if err := json.Unmarshal([]byte(s.Raw), &rec.Fields); err != nil {
		return nil, fmt.Errorf("failed to decode sample with codec %q (registered: %v), plain JSON fallback also failed: %s",
			codecName, recordCodecNames(), err)
	}
	return rec, nil
}

// ─── ai-generate ────────────────────────────────────────────────────────────

type aiGenerateRequestIn struct {
	Generator    string                   `json:"generator"`
	Documents    []string                 `json:"documents"`
	Requirements string                   `json:"requirements"`
	Mode         string                   `json:"mode"`
	CurrentChain []map[string]interface{} `json:"current_chain"`
}

func (h *StudioAPIHandler) aiGenerate(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body := aiGenerateRequestIn{}
	if req.ContentLength != 0 {
		if err := h.DecodeJSON(req, &body); err != nil {
			h.Error400(w, err.Error())
			return
		}
	}

	name := body.Generator
	if name == "" {
		name = "coco"
	}
	generator, ok := pipemod.GetChainGenerator(name)
	if !ok {
		h.WriteError(w, fmt.Sprintf("no chain generator registered under %q", name), http.StatusNotFound)
		return
	}

	docs := make([]util.MapStr, 0, len(body.Documents))
	for _, raw := range body.Documents {
		var m util.MapStr
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			h.Error400(w, fmt.Sprintf("sample document is not a JSON object: %s", err))
			return
		}
		docs = append(docs, m)
	}

	mode := body.Mode
	if mode == "" {
		mode = "generate"
	}

	chain, err := generator.Generate(pipemod.ChainGenerateRequest{
		Context:      req.Context(),
		Mode:         mode,
		Documents:    docs,
		Requirements: body.Requirements,
		CurrentChain: body.CurrentChain,
	})
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if chain == nil {
		chain = []map[string]interface{}{}
	}
	h.WriteJSON(w, util.MapStr{"chain": chain, "generator": generator.Name()}, http.StatusOK)
}

// ─── codecs ─────────────────────────────────────────────────────────────────

func (h *StudioAPIHandler) listCodecs(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	names := recordCodecNames()
	codecs := make([]util.MapStr, 0, len(names))
	for _, n := range names {
		codecs = append(codecs, util.MapStr{"name": n})
	}
	h.WriteJSON(w, util.MapStr{"codecs": codecs}, http.StatusOK)
}
