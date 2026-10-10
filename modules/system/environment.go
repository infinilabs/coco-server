/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/pipeline"
)

// External helpers the document-processing pipeline shells out to. All checks
// run against the coco server's own host — these binaries are exec'd by the
// processors, so "installed on the server" is the only thing that counts.

const (
	envProbeHTTPTimeout = 3 * time.Second
	envProbeCmdTimeout  = 10 * time.Second
)

type environmentCheck struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Status  string `json:"status"` // ok | warning | error
	Version string `json:"version,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

type tikaPipelineOverride struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type effectiveTikaSettings struct {
	Endpoint              string                 `json:"endpoint"`
	TimeoutInSeconds      int                    `json:"timeout_in_seconds"`
	OverriddenByPipelines []tikaPipelineOverride `json:"overridden_by_pipelines,omitempty"`
}

type environmentCheckResult struct {
	Checks        []environmentCheck    `json:"checks"`
	EffectiveTika effectiveTikaSettings `json:"effective_tika"`
}

func (h *APIHandler) checkEnvironment(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	appCfg := common.AppConfig()
	endpoint := appCfg.DocumentProcessing.EffectiveTikaEndpoint()
	timeout := appCfg.DocumentProcessing.EffectiveTikaTimeoutInSeconds()

	result := environmentCheckResult{
		Checks: []environmentCheck{
			checkElasticsearch(),
			checkTika(endpoint),
			checkBinary("libreoffice", "LibreOffice (soffice)",
				[]string{"soffice", "/Applications/LibreOffice.app/Contents/MacOS/soffice"},
				[]string{"--version"},
				"Office documents (docx/pptx/xls) cannot be converted for preview covers.",
				"brew install --cask libreoffice  |  apt-get install libreoffice"),
			checkBinary("pdftoppm", "poppler (pdftoppm)",
				[]string{"pdftoppm"},
				[]string{"-v"},
				"PDF documents cannot be rendered into preview covers.",
				"brew install poppler  |  apt-get install poppler-utils"),
			checkBinary("chrome", "Chrome (headless)",
				[]string{
					"google-chrome", "chromium", "chromium-browser",
					"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
					"/Applications/Chromium.app/Contents/MacOS/Chromium",
					"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
					"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
				},
				[]string{"--version"},
				"Markdown/HTML documents cannot be rendered into preview covers.",
				"brew install --cask google-chrome  |  apt-get install google-chrome-stable"),
		},
		EffectiveTika: effectiveTikaSettings{
			Endpoint:              endpoint,
			TimeoutInSeconds:      timeout,
			OverriddenByPipelines: pipelineTikaOverrides(),
		},
	}

	h.WriteJSON(w, result, http.StatusOK)
}

// checkElasticsearch summarizes the health of the configured engine clusters.
func checkElasticsearch() environmentCheck {
	check := environmentCheck{Key: "elasticsearch", Name: "Elasticsearch"}

	found := false
	var unavailable, red []string
	var healthy []string
	elastic.WalkMetadata(func(key, value interface{}) bool {
		metadata, ok := value.(*elastic.ElasticsearchMetadata)
		if !ok || metadata == nil {
			return true
		}
		found = true
		name := fmt.Sprintf("%v", key)
		switch {
		case !metadata.IsAvailable():
			unavailable = append(unavailable, name)
		case metadata.Health != nil && metadata.Health.Status == "red":
			red = append(red, name)
		default:
			status := "unknown"
			if metadata.Health != nil {
				status = metadata.Health.Status
			}
			healthy = append(healthy, fmt.Sprintf("%s (%s)", name, status))
		}
		return true
	})

	switch {
	case !found:
		check.Status = "error"
		check.Detail = "no elasticsearch cluster configured"
		check.Hint = "configure the elastic connection in coco.yml and restart"
	case len(unavailable) > 0 || len(red) > 0:
		check.Status = "error"
		check.Detail = fmt.Sprintf("unavailable: %v, red: %v, healthy: %v", unavailable, red, healthy)
	default:
		check.Status = "ok"
		check.Detail = strings.Join(healthy, ", ")
	}
	return check
}

// checkTika probes the Tika server's /version endpoint with the effective
// address — the same resolution the extraction processors use.
func checkTika(endpoint string) environmentCheck {
	check := environmentCheck{Key: "tika", Name: "Apache Tika", Detail: endpoint}

	client := &http.Client{Timeout: envProbeHTTPTimeout}
	resp, err := client.Get(strings.TrimRight(endpoint, "/") + "/version")
	if err != nil {
		check.Status = "error"
		check.Detail = fmt.Sprintf("%s: %v", endpoint, err)
		check.Hint = "docker run -d --name tika -p 9998:9998 apache/tika:latest-full"
		return check
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	if resp.StatusCode != http.StatusOK {
		check.Status = "error"
		check.Detail = fmt.Sprintf("%s: returned %d: %s", endpoint, resp.StatusCode, strings.TrimSpace(string(body)))
		check.Hint = "docker run -d --name tika -p 9998:9998 apache/tika:latest-full"
		return check
	}

	check.Status = "ok"
	check.Version = strings.TrimSpace(string(body))
	return check
}

// checkBinary locates a helper binary among the given candidates (PATH names
// then well-known absolute paths) and asks it for a version string. Missing
// helpers degrade cover generation only, hence warning rather than error.
func checkBinary(key, name string, candidates, versionArgs []string, missingImpact, hint string) environmentCheck {
	check := environmentCheck{Key: key, Name: name}

	path := ""
	for _, candidate := range candidates {
		if resolved, err := exec.LookPath(candidate); err == nil {
			path = resolved
			break
		}
	}
	if path == "" {
		check.Status = "warning"
		check.Detail = "not found on the server host. " + missingImpact
		check.Hint = hint
		return check
	}

	ctx, cancel := context.WithTimeout(context.Background(), envProbeCmdTimeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, path, versionArgs...).CombinedOutput()
	if err != nil && len(output) == 0 {
		// present but broken — still better than missing, keep it a warning
		check.Status = "warning"
		check.Detail = fmt.Sprintf("found at %s but failed to run: %v", path, err)
		return check
	}

	check.Status = "ok"
	check.Detail = path
	check.Version = firstNonEmptyLine(string(output))
	return check
}

func firstNonEmptyLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

// pipelineTikaOverrides lists pipelines that pin tika_endpoint explicitly —
// those ignore the global setting until the override is cleared, so the UI
// can call them out instead of silently changing nothing.
func pipelineTikaOverrides() []tikaPipelineOverride {
	ctx := orm.NewContextWithParent(context.Background())
	orm.WithModel(ctx, &pipeline.PipelineConfigV2{})

	configs := []pipeline.PipelineConfigV2{}
	if _, err := elastic.SearchV2WithResultItemMapper(ctx, &configs, orm.NewQuery().Size(1000), nil); err != nil {
		log.Warnf("environment check: failed to list pipelines for tika overrides: %v", err)
		return nil
	}

	var overrides []tikaPipelineOverride
	for _, cfg := range configs {
		if pipelinePinsTika(cfg.Processors) {
			overrides = append(overrides, tikaPipelineOverride{ID: cfg.ID, Name: cfg.Name})
		}
	}
	return overrides
}

func pipelinePinsTika(processors []map[string]interface{}) bool {
	for _, processorDict := range processors {
		for _, rawConfig := range processorDict {
			processorConfig, ok := rawConfig.(map[string]interface{})
			if !ok {
				continue
			}
			if endpoint, ok := processorConfig["tika_endpoint"].(string); ok && strings.TrimSpace(endpoint) != "" {
				return true
			}
		}
	}
	return false
}
