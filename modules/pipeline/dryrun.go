/* Copyright © INFINI LTD.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package pipeline

import (
	"context"
	"fmt"
	"net/http"

	"infini.sh/coco/core"
	"infini.sh/framework/core/api/router"
	"infini.sh/framework/core/config"
	fpipeline "infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const maxStudioDocuments = 10

// stepSnapshot captures the document stream state right after one processor
// ran — the studio's debug view renders per-step field diffs from these.
type stepSnapshot struct {
	Name string `json:"name"`
	// Fields is the first message parsed back into an object after the step.
	Fields util.MapStr `json:"fields,omitempty"`
	// Count is how many messages remain — splitters can expand one document
	// into several, and a step that empties the stream shows count 0.
	Count    int    `json:"count"`
	ErrorStr string `json:"error,omitempty"`
}

// buildNamedProcessor compiles one chain entry with the same semantics as
// pipeline.NewPipeline: an "if" key becomes a conditional (whose then/else
// sub-chains are built by NewIfElseThenProcessor), anything else must be a
// single-key map naming a registered processor.
func buildNamedProcessor(entry map[string]interface{}) (fpipeline.Processor, error) {
	if _, ok := entry["if"]; ok {
		cfg, err := config.NewConfigFrom(entry)
		if err != nil {
			return nil, err
		}
		return fpipeline.NewIfElseThenProcessor(cfg)
	}
	if len(entry) != 1 {
		return nil, fmt.Errorf("processor entry must be a single-key map (or if/then/else), got %d keys", len(entry))
	}
	for procName, cfgRaw := range entry {
		ctor := fpipeline.LookupProcessorConstructor(procName)
		if ctor == nil {
			return nil, fmt.Errorf("unknown processor: %v", procName)
		}
		cfg, err := config.NewConfigFrom(cfgRaw)
		if err != nil {
			return nil, err
		}
		return ctor(cfg)
	}
	return nil, fmt.Errorf("empty processor entry")
}

// runProcessor executes one processor and converts a panic into a step error
// instead of taking the whole HTTP request down with it.
func runProcessor(pctx *fpipeline.Context, proc fpipeline.Processor) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return proc.Process(pctx)
}

// snapshotMessages reads the current document stream from the pipeline
// context. The first message is parsed back into an object for the field
// diff; unparseable payloads fall back to a _raw preview.
func snapshotMessages(pctx *fpipeline.Context) (util.MapStr, int) {
	msgs, _ := pctx.Get(core.PipelineContextDocuments).([]queue.Message)
	if len(msgs) == 0 {
		return nil, 0
	}
	var fields util.MapStr
	if err := util.FromJSONBytes(msgs[0].Data, &fields); err != nil {
		preview := string(msgs[0].Data)
		if len(preview) > 512 {
			preview = preview[:512] + "…"
		}
		fields = util.MapStr{"_raw": preview}
	}
	// Parsed straight from bytes — a fresh object each time, so snapshots
	// never alias mutations made by later processors.
	return fields, len(msgs)
}

// replayChainTraced runs the chain over every sample document, mirroring the
// exact execution contract of process_documents: the document JSON travels
// as a queue.Message under the "messages" context key, and the chain runs
// synchronously in the caller's goroutine. Chain entries execute one at a
// time so a snapshot can be taken between steps.
func replayChainTraced(ctx context.Context, processors []map[string]interface{}, documents []util.MapStr) ([][]stepSnapshot, []util.MapStr, error) {
	type namedProc struct {
		name string
		proc fpipeline.Processor
	}
	procs := make([]namedProc, 0, len(processors))
	for _, entry := range processors {
		inner, err := buildNamedProcessor(entry)
		if err != nil {
			return nil, nil, err
		}
		name := "if/then/else"
		if len(entry) == 1 {
			for k := range entry {
				name = k
			}
		}
		procs = append(procs, namedProc{name: name, proc: inner})
	}

	stepsAll := make([][]stepSnapshot, 0, len(documents))
	finals := make([]util.MapStr, 0, len(documents))
	for _, doc := range documents {
		pctx := fpipeline.AcquireContext(fpipeline.PipelineConfigV2{})
		// Carry the HTTP request context so client cancellation (and with it
		// the server shutdown signal) propagates into processors that
		// consult the embedded context.
		pctx.Context = ctx
		pctx.Set(core.PipelineContextDocuments, []queue.Message{{Data: util.MustToJSONBytes(doc)}})
		// Synchronous callers do not rely on runtime lifecycle semantics,
		// but the context must be marked started for processors to run
		// (same as process_documents).
		pctx.Started()

		steps := make([]stepSnapshot, 0, len(procs))
		var lastFields util.MapStr
		for _, np := range procs {
			snap := stepSnapshot{Name: np.name}
			if err := runProcessor(pctx, np.proc); err != nil {
				snap.ErrorStr = err.Error()
			}
			fields, count := snapshotMessages(pctx)
			snap.Fields = fields
			snap.Count = count
			lastFields = fields
			steps = append(steps, snap)
		}
		stepsAll = append(stepsAll, steps)
		finals = append(finals, lastFields)
	}
	return stepsAll, finals, nil
}

// testPipelineChain — POST /pipeline/studio/test.
// body: {processor: [...], documents: [{...}, ...]} →
// {steps: [[{name, fields, count, error}, ...], ...], finals: [...]}.
// Stateless: nothing is queued or persisted, and the chain never touches the
// real indexing path.
func (h APIHandler) testPipelineChain(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var body struct {
		Processor []map[string]interface{} `json:"processor"`
		Documents []util.MapStr            `json:"documents"`
	}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(body.Documents) == 0 {
		h.WriteError(w, "documents (non-empty object array) is required", http.StatusBadRequest)
		return
	}
	if len(body.Documents) > maxStudioDocuments {
		h.WriteError(w, fmt.Sprintf("at most %d documents per test", maxStudioDocuments), http.StatusBadRequest)
		return
	}

	steps, finals, err := replayChainTraced(req.Context(), body.Processor, body.Documents)
	if err != nil {
		h.WriteError(w, "chain invalid: "+err.Error(), http.StatusBadRequest)
		return
	}
	h.WriteJSON(w, util.MapStr{
		"steps":  steps,
		"finals": finals,
	}, http.StatusOK)
}
