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
 *                    (feed them to pipeline-migrate)
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
	"os"

	// silence must initialize first — see the package doc
	"infini.sh/coco/cmd/internal/pipescan"
	"infini.sh/coco/cmd/pipeline-audit/silence"
	log "infini.sh/framework/core/log"
	"infini.sh/framework/core/pipeline"
)

const (
	verdictCompliant = "compliant"
	verdictMigrate   = "needs-migration"
	verdictInvalid   = "non-compliant"
)

type finding struct {
	Processor string `json:"processor,omitempty"`
	Message   string `json:"message"`
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

	pipescan.RegisterCLIEnvironment()

	var configs []pipescan.SourcedConfig
	if files := flag.Args(); len(files) > 0 {
		for _, path := range files {
			cfg, err := pipescan.LoadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "pipeline-audit: %v\n", err)
				os.Exit(2)
			}
			configs = append(configs, pipescan.SourcedConfig{ID: path, Cfg: cfg})
		}
	} else {
		loaded, err := pipescan.LoadFromES(*esURL, *index, *esToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pipeline-audit: %v\n", err)
			os.Exit(2)
		}
		configs = loaded
	}
	if len(configs) == 0 {
		fmt.Fprintln(os.Stderr, "pipeline-audit: no pipeline configs found (nothing to audit)")
		os.Exit(2)
	}

	results := make([]result, 0, len(configs))
	for _, s := range configs {
		results = append(results, audit(s.Cfg, s.ID))
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
		if r.Verdict != verdictCompliant {
			os.Exit(1)
		}
	}
}

func audit(cfg pipeline.PipelineConfigV2, id string) result {
	r := result{ID: id, Name: cfg.Name}
	if cfg.StrictContract != nil && *cfg.StrictContract {
		r.StrictContract = "true"
	} else {
		r.StrictContract = "off (warn-only)"
	}

	errs, warns := pipescan.Validate(&cfg)
	for _, e := range errs {
		r.Errors = append(r.Errors, finding{Processor: e.Processor, Message: e.Message})
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
	for _, st := range procs {
		cfg, ok := st["process_documents"].(map[string]interface{})
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

/* ---------------- report ---------------- */

func printReport(results []result) {
	counts := map[string]int{}
	fmt.Printf("pipeline-audit — P8 pre-flight (unified pipeline)\n")
	fmt.Printf("scanned %d pipeline config(s)\n\n", len(results))
	for _, r := range results {
		counts[r.Verdict]++
		fmt.Printf("  %-18s %s (%s) — strict_contract: %s\n", "["+r.Verdict+"]", r.Name, r.ID, r.StrictContract)
		for _, e := range r.Errors {
			fmt.Printf("      error:   %s\n", e.Message)
		}
		for _, wn := range r.Warnings {
			fmt.Printf("      warn:    %s\n", wn.Message)
			fmt.Printf("               → pipeline-migrate rewrites this chain; or fix the placement\n")
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
