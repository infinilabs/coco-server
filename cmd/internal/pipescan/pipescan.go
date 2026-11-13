/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infinilabs.com */

/* pipescan is the shared engine of the pipeline CLI tools (pipeline-audit,
 * pipeline-migrate): it boots the full processor catalog outside a server,
 * loads PipelineConfigV2 rows from Elasticsearch or files, and runs the
 * control plane's own compile gate over them. */

package pipescan

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	_ "infini.sh/coco/core/codecs" // register the coco_document/coco_attachment record codecs
	_ "infini.sh/coco/plugins"     // the full coco processor catalog
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/pipeline"
	pipmod "infini.sh/framework/modules/pipeline"
	_ "infini.sh/framework/plugins" // framework processors (http, bulk, smtp, ...)
)

// SourcedConfig is one pipeline config plus where it came from.
type SourcedConfig struct {
	ID       string
	Raw      map[string]interface{} // full stored document (write-back keeps every field)
	FromFile bool
	Cfg      pipeline.PipelineConfigV2
}

// RegisterCLIEnvironment replicates what a server process sets up before
// any chain compiles: the module-setup registrations (dag/echo/for_each
// live in PipeModule.Setup, which a CLI never runs) and an in-memory kv
// store for constructors that touch one at build time (dedup handlers,
// cover caches) — the tools compile chains, they never run them.
func RegisterCLIEnvironment() {
	pipeline.RegisterProcessorPlugin("dag", pipeline.NewDAGProcessor)
	pipeline.RegisterProcessorPlugin("echo", pipmod.NewEchoProcessor)
	pipeline.RegisterDomainProcessorWithConfigMetadata("event", "for_each", pipmod.NewForEachProcessor, pipmod.ForEachConfig{})
	kv.Register("pipeline-cli-mem", memKV{data: map[string]string{}})
}

// Validate compiles the config through the control plane's gate:
// errs are hard rejections, warns are contract mismatches that only
// bite once strict_contract flips on (the P8 signal).
func Validate(cfg *pipeline.PipelineConfigV2) (errs, warns []pipmod.ConfigError) {
	return pipmod.ValidatePipelineConfig(cfg)
}

// LoadFromES pulls stored pipeline rows (raw documents kept for write-back).
func LoadFromES(base, index, token string) ([]SourcedConfig, error) {
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
	var out struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID  string          `json:"_id"`
				Src json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	configs := make([]SourcedConfig, 0, len(out.Hits.Hits))
	for _, h := range out.Hits.Hits {
		var raw map[string]interface{}
		if err := json.Unmarshal(h.Src, &raw); err != nil {
			return nil, fmt.Errorf("decode pipeline %s: %w", h.ID, err)
		}
		cfg, err := configFromMap(raw)
		if err != nil {
			return nil, fmt.Errorf("decode pipeline %s: %w", h.ID, err)
		}
		if cfg.Name == "" {
			cfg.Name = h.ID
		}
		configs = append(configs, SourcedConfig{ID: h.ID, Raw: raw, Cfg: cfg})
	}
	if out.Hits.Total.Value > len(configs) {
		fmt.Fprintf(os.Stderr, "warning: %d stored pipelines, scanned first %d\n",
			out.Hits.Total.Value, len(configs))
	}
	return configs, nil
}

// LoadFile reads one pipeline config: .json directly, anything else via
// YAML into the config-tag shape.
func LoadFile(path string) (pipeline.PipelineConfigV2, error) {
	var cfg pipeline.PipelineConfigV2
	raw, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if strings.HasSuffix(path, ".json") {
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err != nil {
			return cfg, fmt.Errorf("%s: %w", path, err)
		}
		out, err := configFromMap(m)
		return out, fmtWrapped(path, err)
	}
	var doc struct {
		Name           string                   `yaml:"name"`
		StrictContract *bool                    `yaml:"strict_contract"`
		Processors     []map[string]interface{} `yaml:"processor"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	return pipeline.PipelineConfigV2{
		Name:           doc.Name,
		StrictContract: doc.StrictContract,
		Processors:     doc.Processors,
	}, nil
}

func fmtWrapped(path string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", path, err)
}

// configFromMap decodes a generic document (ES _source) into the config
// struct via JSON round-trip.
func configFromMap(m map[string]interface{}) (pipeline.PipelineConfigV2, error) {
	var cfg pipeline.PipelineConfigV2
	b, err := json.Marshal(m)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Processors == nil {
		if raw, ok := m["processor"].([]interface{}); ok {
			cfg.Processors = make([]map[string]interface{}, 0, len(raw))
			for _, item := range raw {
				if step, ok := item.(map[string]interface{}); ok {
					cfg.Processors = append(cfg.Processors, step)
				}
			}
		}
	}
	return cfg, nil
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
