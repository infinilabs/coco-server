/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infinilabs.com */

/* pipeline-migrate — the P8 one-shot migration (unified pipeline
 * WORKPLAN #114): rewrites legacy chains whose record processors sit at
 * the batch root into the record convention:
 *
 *   processor: [{X:{}}, {Y:{}}]                (legacy messages shape)
 *     → processor:
 *       - for_each:
 *           codec: coco_document
 *           processor: [{X:{}}, {Y:{}}}]
 *
 * Only chains the pipeline-audit would flag are touched (contract
 * warnings, no compile errors); the migrated shape is validated through
 * the control plane gate and must come back fully compliant (zero
 * errors, zero warnings) before it counts. Chains mixing batch-scoped
 * steps are refused — segmentation there needs a human.
 *
 * Dry-run by default: prints the migration. --write updates the stored
 * documents in Elasticsearch directly (full document write-back, other
 * fields untouched). Exit 0 = nothing to do or all migrated; 1 = any
 * refusal or failure; 2 = operational errors. */

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	// silence must initialize first — see the package doc
	"infini.sh/coco/cmd/internal/pipescan"
	"infini.sh/coco/cmd/pipeline-audit/silence"
	log "infini.sh/framework/core/log"
	"infini.sh/framework/core/pipeline"
	pipmod "infini.sh/framework/modules/pipeline"
)

func main() {
	esURL := flag.String("es", "http://127.0.0.1:9205", "Elasticsearch base URL (stored pipelines)")
	index := flag.String("index", "coco_pipelineconfigv2", "pipeline config index")
	esToken := flag.String("es-token", "", "optional bearer token for Elasticsearch")
	write := flag.Bool("write", false, "update stored documents (default: dry-run, print only)")
	var only stringSlice
	flag.Var(&only, "only", "restrict to these pipeline ids (repeatable)")
	jsonOut := flag.Bool("json", false, "machine-readable report on stdout")
	flag.Parse()

	pipescan.RegisterCLIEnvironment()
	esConfig.base, esConfig.index, esConfig.token = *esURL, *index, *esToken
	doWrite = *write

	var configs []pipescan.SourcedConfig
	if files := flag.Args(); len(files) > 0 {
		for _, path := range files {
			cfg, err := pipescan.LoadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "pipeline-migrate: %v\n", err)
				os.Exit(2)
			}
			configs = append(configs, pipescan.SourcedConfig{ID: path, FromFile: true, Cfg: cfg})
		}
	} else {
		loaded, err := pipescan.LoadFromES(*esURL, *index, *esToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pipeline-migrate: %v\n", err)
			os.Exit(2)
		}
		configs = loaded
	}
	if len(only) > 0 {
		wanted := map[string]bool{}
		for _, id := range only {
			wanted[id] = true
		}
		filtered := configs[:0]
		for _, s := range configs {
			if wanted[s.ID] {
				filtered = append(filtered, s)
			}
		}
		configs = filtered
	}
	if len(configs) == 0 {
		fmt.Fprintln(os.Stderr, "pipeline-migrate: no matching pipeline configs")
		os.Exit(2)
	}

	results := make([]result, 0, len(configs))
	for _, s := range configs {
		results = append(results, migrateOne(s))
	}

	// drain async seelog buffers while still silenced, then restore
	log.Flush()
	silence.End()

	if *jsonOut {
		out, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(out))
	} else {
		printReport(results)
	}

	for _, r := range results {
		if r.Outcome != outcomeMigrated && r.Outcome != outcomeNothingToDo && r.Outcome != outcomeSkipped {
			os.Exit(1)
		}
	}
}

const (
	outcomeMigrated    = "migrated"
	outcomeNothingToDo = "already-compliant"
	outcomeSkipped     = "skipped"
	outcomeRefused     = "refused"
	outcomeFailed      = "failed"
)

type result struct {
	ID      string                   `json:"id"`
	Name    string                   `json:"name"`
	Outcome string                   `json:"outcome"`
	Reason  string                   `json:"reason,omitempty"`
	Before  []map[string]interface{} `json:"before,omitempty"`
	After   []map[string]interface{} `json:"after,omitempty"`
	Written bool                     `json:"written,omitempty"`
}

func migrateOne(s pipescan.SourcedConfig) result {
	r := result{ID: s.ID, Name: s.Cfg.Name}

	errs, warns := pipescan.Validate(&s.Cfg)
	if len(errs) > 0 {
		r.Outcome = outcomeSkipped
		r.Reason = fmt.Sprintf("does not compile (%d error(s), first: %s) — fix before migrating", len(errs), errs[0].Message)
		return r
	}
	if len(warns) == 0 {
		r.Outcome = outcomeNothingToDo
		r.Reason = "no contract warnings — nothing to migrate"
		return r
	}

	after, err := migrateChain(s.Cfg.Processors)
	if err != nil {
		r.Outcome = outcomeRefused
		r.Reason = err.Error()
		return r
	}

	// the migrated shape must validate fully clean — errors OR warnings
	// mean the rewrite is wrong, never write it
	migrated := s.Cfg
	migrated.Processors = after
	mErrs, mWarns := pipescan.Validate(&migrated)
	if len(mErrs) > 0 || len(mWarns) > 0 {
		r.Outcome = outcomeFailed
		r.Reason = fmt.Sprintf("migrated shape did not validate clean (errs=%d warns=%d, first: %v)",
			len(mErrs), len(mWarns), firstMessage(mErrs, mWarns))
		return r
	}

	r.Outcome = outcomeMigrated
	r.Before = s.Cfg.Processors
	r.After = after

	if doWrite {
		if s.FromFile {
			r.Outcome = outcomeFailed
			r.Reason = "write-back is ES-only; file inputs stay dry-run"
			return r
		}
		if err := writeDoc(s.ID, s.Raw, after); err != nil {
			r.Outcome = outcomeFailed
			r.Reason = "write-back failed: " + err.Error()
			r.Written = false
			return r
		}
		r.Written = true
	}
	return r
}

func firstMessage(errs, warns []pipmod.ConfigError) string {
	if len(errs) > 0 {
		return errs[0].Message
	}
	if len(warns) > 0 {
		return warns[0].Message
	}
	return ""
}

/* ---------------- transform ---------------- */

// namespaceCodecs maps record namespaces to their codecs. Everything
// else refuses — an unmapped namespace needs a codec the tool doesn't
// know, a human should place it.
var namespaceCodecs = map[string]string{
	"document":   "coco_document",
	"attachment": "coco_attachment",
}

// migrateChain rewrites a legacy root chain into the record convention.
// Steps are grouped by their declared namespace into maximal contiguous
// runs; each run becomes one for_each host with the matching codec
// (unconstrained steps inherit the neighbor's namespace — they ran
// per-message under the legacy host too). Batch-scoped steps refuse:
// segmentation there changes semantics and needs a human.
func migrateChain(procs []map[string]interface{}) ([]map[string]interface{}, error) {
	if len(procs) == 0 {
		return nil, fmt.Errorf("empty chain")
	}

	names := make([]string, len(procs))
	for i, step := range procs {
		name := stepName(step)
		contract := pipeline.RecordContractOf(name)
		if contract.Scope == pipeline.ScopeBatch {
			return nil, fmt.Errorf("chain mixes batch-scoped steps (%s) with record processors — manual segmentation required", name)
		}
		if contract.Namespace != "" {
			if _, known := namespaceCodecs[contract.Namespace]; !known {
				return nil, fmt.Errorf("processor [%s] requires namespace [%s] which has no known codec — manual migration required", name, contract.Namespace)
			}
			names[i] = contract.Namespace
		}
	}
	// unconstrained steps inherit forward, then backward for a leading run
	last := ""
	for i := range names {
		if names[i] == "" {
			names[i] = last
		} else {
			last = names[i]
		}
	}
	next := ""
	for i := len(names) - 1; i >= 0; i-- {
		if names[i] == "" {
			names[i] = next
		} else {
			next = names[i]
		}
	}

	var out []map[string]interface{}
	start := 0
	for i := 1; i <= len(names); i++ {
		if i == len(names) || names[i] != names[start] {
			out = append(out, map[string]interface{}{
				"for_each": map[string]interface{}{
					"codec":     namespaceCodecs[names[start]],
					"processor": cloneSteps(procs[start:i]),
				},
			})
			start = i
		}
	}
	return out, nil
}

// cloneSteps deep-copies the step maps so the migrated document never
// shares structures with the loaded one.
func cloneSteps(procs []map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, len(procs))
	for i, step := range procs {
		b, _ := json.Marshal(step)
		var m map[string]interface{}
		_ = json.Unmarshal(b, &m)
		out[i] = m
	}
	return out
}

func stepName(step map[string]interface{}) string {
	for k := range step {
		return k
	}
	return ""
}

/* ---------------- write-back ---------------- */

var (
	esConfig = struct{ base, index, token string }{}
	doWrite  bool
)

func writeDoc(id string, raw map[string]interface{}, procs []map[string]interface{}) error {
	doc := make(map[string]interface{}, len(raw)+1)
	for k, v := range raw {
		doc[k] = v
	}
	doc["processor"] = procs
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut,
		strings.TrimRight(esConfig.base, "/")+"/"+esConfig.index+"/_doc/"+id+"?refresh=wait_for",
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if esConfig.token != "" {
		req.Header.Set("Authorization", "Bearer "+esConfig.token)
	}
	res, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	rb, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return fmt.Errorf("elasticsearch %s: %s", res.Status, string(rb))
	}
	return nil
}

/* ---------------- report ---------------- */

func printReport(results []result) {
	counts := map[string]int{}
	fmt.Printf("pipeline-migrate — P8 one-shot migration (unified pipeline)\n")
	fmt.Printf("scanned %d pipeline config(s)\n\n", len(results))
	for _, r := range results {
		counts[r.Outcome]++
		fmt.Printf("  [%s] %s (%s)\n", r.Outcome, r.Name, r.ID)
		if r.Reason != "" {
			fmt.Printf("      %s\n", r.Reason)
		}
		if r.Outcome == outcomeMigrated {
			before, _ := yaml.Marshal(r.Before)
			after, _ := yaml.Marshal(r.After)
			fmt.Printf("      before:\n%s", indent(string(before)))
			fmt.Printf("      after:\n%s", indent(string(after)))
			if r.Written {
				fmt.Printf("      written back to elasticsearch\n")
			} else {
				fmt.Printf("      (dry-run — add --write to update the stored document)\n")
			}
		}
	}
	fmt.Printf("\nsummary: %d scanned | %d migrated | %d already compliant | %d skipped | %d refused | %d failed\n",
		len(results), counts[outcomeMigrated], counts[outcomeNothingToDo], counts[outcomeSkipped], counts[outcomeRefused], counts[outcomeFailed])
}

func indent(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = "        " + l
	}
	return strings.Join(lines, "\n") + "\n"
}

// stringSlice is a repeatable string flag.
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }

func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}
