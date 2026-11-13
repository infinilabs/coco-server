/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
)

// fetchModelsRequest carries the connection details typed into the provider
// form; the vendor model-list endpoint is queried with the same credentials
// the provider will be saved with.
type fetchModelsRequest struct {
	APIType string `json:"api_type"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

// fetchModels lists the model ids a vendor offers (OpenAI-compatible
// GET /models, ollama GET /api/tags) so the add-model modal can offer real
// choices instead of free-text entry. Proxied through the backend so browser
// CORS rules never produce false negatives and the api key never leaves the
// server side. Fetching is user-triggered and purely advisory: failures fall
// back to manual entry without blocking the modal.
func (h *APIHandler) fetchModels(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var r fetchModelsRequest
	if err := h.DecodeJSON(req, &r); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	r.BaseURL = strings.TrimSuffix(strings.TrimSpace(r.BaseURL), "/")
	if r.BaseURL == "" {
		h.WriteError(w, "base_url is required", http.StatusBadRequest)
		return
	}

	url := listModelsURL(r)
	if url == "" {
		h.WriteError(w, "unsupported api_type: "+r.APIType, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(req.Context(), 15*time.Second)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		h.WriteJSON(w, util.MapStr{"ok": false, "error": err.Error()}, http.StatusOK)
		return
	}
	if r.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+r.APIKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		h.WriteJSON(w, util.MapStr{"ok": false, "error": err.Error()}, http.StatusOK)
		return
	}
	defer resp.Body.Close()

	// model lists can be sizeable (200+ entries on some vendors); cap the read
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := strings.TrimSpace(string(body))
		if len(detail) > 300 {
			detail = detail[:300] + "..."
		}
		h.WriteJSON(w, util.MapStr{"ok": false, "status": resp.StatusCode, "error": detail}, http.StatusOK)
		return
	}

	ids := parseModelList(strings.EqualFold(r.APIType, "ollama"), body)
	if ids == nil {
		ids = []string{}
	}
	h.WriteJSON(w, util.MapStr{"ok": true, "models": ids}, http.StatusOK)
}

// listModelsURL resolves the vendor model-list endpoint. OpenAI-compatible
// vendors frequently omit the /v1 prefix when entering a base url, so add it
// unless the caller already included one.
func listModelsURL(r fetchModelsRequest) string {
	if strings.EqualFold(r.APIType, "ollama") {
		return r.BaseURL + "/api/tags"
	}
	if strings.EqualFold(r.APIType, "openai") {
		if !strings.HasSuffix(r.BaseURL, "/v1") && !strings.Contains(r.BaseURL, "/v1/") {
			return r.BaseURL + "/v1/models"
		}
		return r.BaseURL + "/models"
	}
	return ""
}

// parseModelList extracts ids from both response shapes; an unknown shape
// returns nil and surfaces as an empty list on the client.
func parseModelList(ollama bool, body []byte) []string {
	var out []string
	if ollama {
		var v struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if json.Unmarshal(body, &v) == nil {
			for _, m := range v.Models {
				if m.Name != "" {
					out = append(out, m.Name)
				}
			}
		}
		return out
	}
	var v struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &v) == nil {
		for _, m := range v.Data {
			if m.ID != "" {
				out = append(out, m.ID)
			}
		}
	}
	return out
}
