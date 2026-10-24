/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infinilabs.com */

/* pipeline-audit — the P8 pre-flight scan (unified pipeline WORKPLAN #113).
 *
 * Compiles every existing pipeline config with the same gate the control
 * plane uses (ValidatePipelineConfig: registry existence, unpacking,
 * nested sub-chains, record-contract placement) and classifies each:
 *
 *   compliant        compiles clean, no contract warnings
 *   needs-migration  compiles today only because strict_contract is off —
 *                    exactly the pipelines the P8 default flip would break
 *   non-compliant    does not compile at all (unknown processor, bad
 *                    config, strict violation when opted in)
 *
 * Sources: --es (stored PipelineConfigV2 rows, default localhost:9205)
 * and/or --file (a single pipeline config in YAML or JSON). Exit 0 when
 * everything is compliant, 1 on findings, 2 on operational errors. */

package main

import (
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
	"infini.sh/coco/cmd/pipeline-audit/silence"

	_ "infini.sh/coco/core/codecs" // register the coco_document/coco_attachment record codecs
	_ "infini.sh/coco/plugins"     // the full coco processor catalog
	"infini.sh/framework/core/kv"
	log "infini.sh/framework/core/log"
	"infini.sh/framework/core/pipeline"
	pipmod "infini.sh/framework/modules/pipeline"
	_ "infini.sh/framework/plugins" // framework processors (http, bulk, smtp, ...)
)

const (
	verdictCompliant = "compliant"
	verdictMigrate   = "needs-migration"
	verdictInvalid   = "non-compliant"
)

type finding struct {
	Pipeline  string `json:"pipeline,omitempty"`
	Processor string `json:"processor,omitempty"`
	Message   string `json:"message"`
}

type sourcedConfig struct {
	id  string
	cfg pipeline.PipelineConfigV2
}

type result struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Verdict        string    `json:"verdict"`
	StrictContract string    `json:"strict_contract"`
	Errors         []finding `json:"errors,omitempty"`
	Warnings       []finding `json:"warnings,omitempty"`
	Notes          []string  `json:"notes,omitempty"`
}

func main() {
	esURL := flag.String("es", "http://127.0.0.1:9205", "Elasticsearch base URL (stored pipelines)")
	index := flag.String("index", "coco_pipelineconfigv2", "pipeline config index")
	esToken := flag.String("es-token", "", "optional bearer token for Elasticsearch")
	jsonOut := flag.Bool("json", false, "machine-readable report on stdout")
	flag.Parse()

	// module-setup registrations the tool must replicate (they live in
	// PipeModule.Setup, which a CLI never runs): dag/echo/for_each
	pipeline.RegisterProcessorPlugin("dag", pipeline.NewDAGProcessor)
	pipeline.RegisterProcessorPlugin("echo", pipmod.NewEchoProcessor)
	pipeline.RegisterDomainProcessorWithConfigMetadata("event", "for_each", pipmod.NewForEachProcessor, pipmod.ForEachConfig{})

	// some coco constructors touch the kv store at build time (dedup
	// handlers, cover caches); the audit compiles chains but never runs
	// them, so a throwaway in-memory store is all they need
	kv.Register("pipeline-audit-mem", memKV{data: map[string]string{}})

	var configs []sourcedConfig

	for _, path := range flag.Args() {
		cfg, err := loadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pipeline-audit: %v\n", err)
			os.Exit(2)
		}
		configs = append(configs, sourcedConfig{id: path, cfg: cfg})
	}
	if *esURL != "" && len(flag.Args()) == 0 {
		loaded, err := loadFromES(*esURL, *index, *esToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pipeline-audit: %v\n", err)
			os.Exit(2)
		}
		configs = append(configs, loaded...)
	}
	if len(configs) == 0 {
		fmt.Fprintln(os.Stderr, "pipeline-audit: no pipeline configs found (nothing to audit)")
		os.Exit(2)
	}

	results := make([]result, 0, len(configs))
	for _, s := range configs {
		results = append(results, audit(s.id, s.cfg))
	}

	// drain async seelog buffers while still silenced, then give the
	// report a clean stdout (no-op when PIPELINE_AUDIT_VERBOSE is set)
	log.Flush()
	silence.End()

	if *jsonOut {
		out, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(out))
	} else {
		printReport(results)
	}

	for _, r := range results {
		if r.Verdict != verdictCompliant {
			os.Exit(1)
		}
	}
}

func audit(id string, cfg pipeline.PipelineConfigV2) result {
	r := result{ID: id, Name: cfg.Name}
	if cfg.StrictContract != nil && *cfg.StrictContract {
		r.StrictContract = "true"
	} else {
		r.StrictContract = "off (warn-only)"
	}

	errs, warns := pipmod.ValidatePipelineConfig(&cfg)
	for _, e := range errs {
		r.Errors = append(r.Errors, finding{Pipeline: e.Processor, Processor: e.Processor, Message: e.Message})
	}
	for _, wn := range warns {
		r.Warnings = append(r.Warnings, finding{Processor: wn.Processor, Message: wn.Message})
	}
	r.Notes = modeNotes(cfg.Processors)

	switch {
	case len(errs) > 0:
		r.Verdict = verdictInvalid
	case len(warns) > 0:
		r.Verdict = verdictMigrate
	default:
		r.Verdict = verdictCompliant
	}
	return r
}

// modeNotes flags process_documents hosts still on the legacy messages
// convention — P8 #116 flips the default to records, these need a visit.
func modeNotes(procs []map[string]interface{}) []string {
	var notes []string
	for _, step := range procs {
		cfg, ok := step["process_documents"].(map[string]interface{})
		if !ok {
			continue
		}
		mode, _ := cfg["mode"].(string)
		if mode != "records" {
			if mode == "" {
				mode = "(unset → messages)"
			}
			notes = append(notes, fmt.Sprintf("process_documents mode=%s — legacy messages convention, records is the P8 target", mode))
		}
	}
	return notes
}

/* ---------------- sources ---------------- */

type esHit struct {
	ID  string          `json:"_id"`
	Src json.RawMessage `json:"_source"`
}

type esResp struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []esHit `json:"hits"`
	} `json:"hits"`
}

func loadFromES(base, index, token string) ([]sourcedConfig, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet,
		strings.TrimRight(base, "/")+"/"+index+"/_search?size=1000", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch unreachable: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("elasticsearch %s: %s", res.Status, truncate(string(body), 200))
	}
	var out esResp
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	configs := make([]sourcedConfig, 0, len(out.Hits.Hits))
	for _, h := range out.Hits.Hits {
		var cfg pipeline.PipelineConfigV2
		if err := json.Unmarshal(h.Src, &cfg); err != nil {
			return nil, fmt.Errorf("decode pipeline %s: %w", h.ID, err)
		}
		if cfg.Name == "" {
			cfg.Name = h.ID
		}
		configs = append(configs, sourcedConfig{id: h.ID, cfg: cfg})
	}
	if out.Hits.Total.Value > len(configs) {
		fmt.Fprintf(os.Stderr, "pipeline-audit: warning: %d stored pipelines, scanned first %d\n",
			out.Hits.Total.Value, len(configs))
	}
	return configs, nil
}

// loadFile reads one pipeline config: .json directly, anything else via
// YAML into the config-tag shape.
func loadFile(path string) (pipeline.PipelineConfigV2, error) {
	var cfg pipeline.PipelineConfigV2
	raw, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if strings.HasSuffix(path, ".json") {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return cfg, fmt.Errorf("%s: %w", path, err)
		}
		return cfg, nil
	}
	var doc struct {
		Name           string                   `yaml:"name"`
		StrictContract *bool                    `yaml:"strict_contract"`
		Processors     []map[string]interface{} `yaml:"processor"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	cfg.Name = doc.Name
	cfg.StrictContract = doc.StrictContract
	cfg.Processors = doc.Processors
	return cfg, nil
}

/* ---------------- report ---------------- */

func printReport(results []result) {
	counts := map[string]int{}
	fmt.Printf("pipeline-audit — P8 pre-flight (unified pipeline)\n")
	fmt.Printf("scanned %d pipeline config(s)\n\n", len(results))
	for _, r := range results {
		counts[r.Verdict]++
		label := fmt.Sprintf("[%s]", r.Verdict)
		fmt.Printf("  %-18s %s (%s) — strict_contract: %s\n", label, r.Name, r.ID, r.StrictContract)
		for _, e := range r.Errors {
			fmt.Printf("      error:   %s\n", e.Message)
		}
		for _, wn := range r.Warnings {
			fmt.Printf("      warn:    %s\n", wn.Message)
			fmt.Printf("               → wrap in for_each{codec: coco_document, processor:[...]} or fix the placement\n")
		}
		for _, n := range r.Notes {
			fmt.Printf("      note:    %s\n", n)
		}
	}
	fmt.Printf("\nsummary: %d scanned | %d compliant | %d needs-migration | %d non-compliant\n",
		len(results), counts[verdictCompliant], counts[verdictMigrate], counts[verdictInvalid])
	if blockers := counts[verdictMigrate] + counts[verdictInvalid]; blockers > 0 {
		fmt.Printf("P8 default-flip blockers: %d\n", blockers)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

/* memKV backs constructor-time kv lookups outside a real server. */
type memKV struct{ data map[string]string }

func (memKV) Open() error  { return nil }
func (memKV) Close() error { return nil }

func memKey(bucket string, key []byte) string { return bucket + "\x00" + string(key) }

func (k memKV) GetValue(bucket string, key []byte) ([]byte, error) {
	if v, ok := k.data[memKey(bucket, key)]; ok {
		return []byte(v), nil
	}
	return nil, fmt.Errorf("not found: %s/%s", bucket, string(key))
}

func (k memKV) GetCompressedValue(bucket string, key []byte) ([]byte, error) {
	return k.GetValue(bucket, key)
}

func (k memKV) AddValue(bucket string, key []byte, value []byte) error {
	k.data[memKey(bucket, key)] = string(value)
	return nil
}

func (k memKV) AddValueCompress(bucket string, key []byte, value []byte) error {
	return k.AddValue(bucket, key, value)
}

func (k memKV) ExistsKey(bucket string, key []byte) (bool, error) {
	_, ok := k.data[memKey(bucket, key)]
	return ok, nil
}

func (k memKV) DeleteKey(bucket string, key []byte) error {
	delete(k.data, memKey(bucket, key))
	return nil
}
