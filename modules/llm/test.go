/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package llm

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
)

// testModelRequest carries the connection details typed into the provider
// form (or read back on the edit page) plus the model being verified.
type testModelRequest struct {
	APIType string `json:"api_type"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
	Type    string `json:"type"`
}

// testModel probes model availability before the provider is saved. It speaks
// just enough of each protocol to verify URL reachability, the API key and
// the model id in one round trip:
//
//	language/vision -> chat completions
//	embedding       -> embeddings
//	rerank          -> rerank (OpenAI-compatible vendors; ollama has none)
//
// ollama uses its native /api/* paths and takes no auth header. The probe is
// proxied through the backend so browser CORS rules never produce false
// negatives.
func (h *APIHandler) testModel(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var r testModelRequest
	if err := h.DecodeJSON(req, &r); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	r.Model = strings.TrimSpace(r.Model)
	r.BaseURL = strings.TrimSuffix(strings.TrimSpace(r.BaseURL), "/")
	if r.Model == "" || r.BaseURL == "" {
		h.WriteError(w, "model and base_url are required", http.StatusBadRequest)
		return
	}

	url, body := buildModelProbeRequest(r)
	if url == "" {
		h.WriteError(w, "unsupported model type: "+r.Type, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(req.Context(), 15*time.Second)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		h.WriteJSON(w, util.MapStr{"ok": false, "error": err.Error()}, http.StatusOK)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if r.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+r.APIKey)
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		h.WriteJSON(w, util.MapStr{"ok": false, "error": err.Error()}, http.StatusOK)
		return
	}
	defer resp.Body.Close()
	elapsed := time.Since(start).Milliseconds()

	// 4xx/5xx bodies carry the vendor's reason (bad key, unknown model);
	// trim them so a huge HTML error page never floods the response.
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	detail := strings.TrimSpace(string(respBody))
	if len(detail) > 300 {
		detail = detail[:300] + "..."
	}

	ok := resp.StatusCode >= 200 && resp.StatusCode < 300
	result := util.MapStr{"ok": ok, "status": resp.StatusCode, "latency_ms": elapsed}
	if ok {
		result["detail"] = detail
	} else {
		result["error"] = detail
	}
	h.WriteJSON(w, result, http.StatusOK)
}

// buildModelProbeRequest picks the probe path and payload per api type and
// model type; returns an empty url for unsupported combinations.
func buildModelProbeRequest(r testModelRequest) (string, []byte) {
	isOllama := strings.EqualFold(r.APIType, "ollama")
	switch strings.ToLower(r.Type) {
	case "embedding":
		if isOllama {
			return r.BaseURL + "/api/embeddings", util.MustToJSONBytes(util.MapStr{"model": r.Model, "prompt": "ping"})
		}
		return r.BaseURL + "/embeddings", util.MustToJSONBytes(util.MapStr{"model": r.Model, "input": []string{"ping"}})
	case "rerank":
		if isOllama {
			return "", nil
		}
		return r.BaseURL + "/rerank", util.MustToJSONBytes(util.MapStr{"model": r.Model, "query": "ping", "documents": []string{"alpha", "beta"}})
	default: // language / vision / unset
		if isOllama {
			return r.BaseURL + "/api/chat", util.MustToJSONBytes(util.MapStr{"model": r.Model, "stream": false, "messages": []util.MapStr{{"role": "user", "content": "ping"}}})
		}
		return r.BaseURL + "/chat/completions", util.MustToJSONBytes(util.MapStr{"model": r.Model, "max_tokens": 1, "messages": []util.MapStr{{"role": "user", "content": "ping"}}})
	}
}
