/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"infini.sh/framework/core/security"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/framework/lib/fasttemplate"
	elastic1 "infini.sh/framework/modules/elastic/common"
	"infini.sh/framework/plugins/replay"
)

type SetupConfig struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	Language string `json:"language,omitempty"`
}

// SetupDefaultModelConfig is the payload for the default-model setup API,
// which lets the user pick default language/vision/embedding models. Every
// selection is optional and the endpoint may be called multiple times; it
// does not affect the overall setup completion state.
type SetupDefaultModelConfig struct {
	LanguageModel  *SetupModelConfig `json:"language_model,omitempty"`
	VisionModel    *SetupModelConfig `json:"vision_model,omitempty"`
	EmbeddingModel *SetupModelConfig `json:"embedding_model,omitempty"`
}

// SetupModelConfig describes a single model selection in the setup wizard.
//
// Two independent dimensions:
//
//  1. Provider: ModelProvider.ID refers to an existing (builtin or previously
//     created) provider; otherwise ModelProvider's other fields define a new
//     custom provider to be created on the fly.
//
//  2. Model: exactly one of ModelID or Model must be set.
//     - ModelID picks an already-registered model on the provider (by name).
//     - Model defines a new model to add to the provider, including whether
//       it supports reasoning mode (Model.SupportReasoning, language models
//       only).
//
// APIToken is the provider-level API token; it is written to the provider
// regardless of which model dimension is used.
type SetupModelConfig struct {
	ModelProvider SetupModelProvider `json:"model_provider,omitempty"`

	ModelID string         `json:"model_id,omitempty"`
	Model   *SetupModelDef `json:"model,omitempty"`

	APIToken string `json:"api_token,omitempty"`
}

// SetupModelDef defines a new model to be added to the selected provider.
type SetupModelDef struct {
	ID string `json:"id"`
	// SupportReasoning indicates whether this model is capable of reasoning
	// mode. Only meaningful for language models; ignored for vision /
	// embedding.
	SupportReasoning bool `json:"support_reasoning,omitempty"`
}

// SetupModelProvider is either a reference (ID only) to a builtin provider,
// or a full custom provider definition. The API token always travels at the
// SetupModelConfig level, not here.
type SetupModelProvider struct {
	ID string `json:"id,omitempty"`

	/*
	 * Fields for a new provider
	 */
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	BaseURL     string `json:"base_url,omitempty"`
	APIType     string `json:"api_type,omitempty"` // "openai" or "ollama"
}

// SetupDoneKey marks server initialization as finished. Once present the
// setup wizard is considered complete and re-running it is rejected.
var SetupDoneKey = ".setup_done"

// isSetupDone reports whether server initialization has finished, and keeps
// the framework-level setup gate in sync.
func isSetupDone() bool {
	exists, err := kv.ExistsKey(core.DefaultSettingBucketKey, []byte(SetupDoneKey))
	done := exists || err != nil
	global.Env().EnableSetup(!done)
	return done
}

// setupInitialize runs the server initialization wizard: create the admin
// user and populate the bundled ES templates (model providers, assistants,
// MCP servers, roles, etc.). The done flag is only written after the ES
// refresh succeeds, so that a partial failure leaves the wizard re-runnable.
func (h *APIHandler) setupInitialize(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	if isSetupDone() {
		panic("setup has already been completed")
	}

	input := SetupConfig{}
	if err := h.DecodeJSON(req, &input); err != nil {
		panic(err)
	}

	info := common.AppConfig()

	if input.Name != "" {
		info.ServerInfo.Name = fmt.Sprintf("%s's Coco Server", input.Name)
	} else if info.ServerInfo.Name == "" {
		info.ServerInfo.Name = "My Coco Server"
	}

	if !global.Env().SystemConfig.WebAppConfig.Security.Managed {
		if info.ServerInfo.Endpoint == "" {
			var schema = "http"
			if req.TLS != nil {
				schema = "https"
			}
			info.ServerInfo.Endpoint = fmt.Sprintf("%s://%s", schema, req.Host)
		}

		if input.Password == "" {
			panic("password can't be empty")
		}

		user, err := security.MustGetAuthenticationProvider(security.DefaultNativeAuthBackend).CreateUser(input.Name, input.Email, input.Password, true)
		if user == nil || user.ID == "" {
			panic("failed to init user")
		}
		if err != nil {
			panic(err)
		}

		//initialize setup templates
		if err := h.initializeSetupTemplates(user.ID, input, info.ServerInfo.Endpoint); err != nil {
			panic(err)
		}
	}

	// Force ES to refresh so that the documents written above are visible to
	// the very next request — the default-model setup API needs to list the
	// freshly inserted model providers via _search.
	if err := refreshSetupIndices(); err != nil {
		panic(fmt.Errorf("refresh setup indices: %w", err))
	}

	//save app config
	common.SetAppConfig(&info)

	// Mark setup as done last so partial failures above leave the wizard
	// re-runnable.
	if err := kv.AddValue(core.DefaultSettingBucketKey, []byte(SetupDoneKey), []byte(time.Now().String())); err != nil {
		panic(err)
	}
	isSetupDone()

	h.WriteAckOKJSON(w)
}

// setupInitializeDefaultModel persists the user's default model selections
// (language/vision/embedding).
//
// NOTE: on the UI the initialization wizard is presented as two steps, but in
// the backend "initialization" is a single step (setupInitialize) — this
// endpoint is an independent default-model setter that does NOT affect the
// overall setup completion state and may be called repeatedly. Keep this UI
// vs. backend mismatch in mind when wiring the wizard.
func (h *APIHandler) setupInitializeDefaultModel(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	input := SetupDefaultModelConfig{}
	if err := h.DecodeJSON(req, &input); err != nil {
		panic(err)
	}

	// Validate the payload before doing any persistence
	if err := validateDefaultModelConfig(&input); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}

	info := common.AppConfig()

	if err := h.applySetupDefaultModels(req, &input, &info); err != nil {
		panic(err)
	}

	// Refresh model-provider index so that any provider just created/updated
	// is immediately visible to subsequent reads.
	if err := refreshModelProviderIndex(); err != nil {
		panic(fmt.Errorf("refresh model-provider index: %w", err))
	}

	common.SetAppConfig(&info)

	h.WriteAckOKJSON(w)
}

// refreshSetupIndices forces an ES refresh on every coco-prefixed index so
// setup's bulk writes are searchable to subsequent reads.
func refreshSetupIndices() error {
	prefix := setupIndexPrefix()
	esClient := elastic.GetClient(global.MustLookupString(elastic.GlobalSystemElasticsearchID))
	return esClient.Refresh(prefix + "*")
}

// refreshModelProviderIndex forces an ES refresh on the model-provider index
// after default-model setup writes provider documents.
func refreshModelProviderIndex() error {
	prefix := setupIndexPrefix()
	esClient := elastic.GetClient(global.MustLookupString(elastic.GlobalSystemElasticsearchID))
	return esClient.Refresh(prefix + "model-provider" + common.GetSchemaSuffix())
}

// setupIndexPrefix returns the configured ES index prefix, defaulting to
// "coco_" when none is set.
func setupIndexPrefix() string {
	cfg := elastic1.ORMConfig{}
	exist, err := env.ParseConfig("elastic.orm", &cfg)
	if exist && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}
	if cfg.IndexPrefix == "" {
		return "coco_"
	}
	return cfg.IndexPrefix
}

func (h *APIHandler) initializeConnector() error {
	var dsl []byte
	baseDir := path.Join(global.Env().GetConfigDir(), "setup")
	dslTplFile := filepath.Join(baseDir, "connector.tpl")
	dsl, err := util.FileGetContent(dslTplFile)
	if err != nil {
		return err
	}
	if len(dsl) == 0 {
		return fmt.Errorf("got empty template [%s]", dslTplFile)
	}

	var tpl *fasttemplate.Template
	tpl, err = fasttemplate.NewTemplate(string(dsl), "$[[", "]]")
	cfg1 := elastic1.ORMConfig{}
	exist, err := env.ParseConfig("elastic.orm", &cfg1)
	if exist && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}

	if cfg1.IndexPrefix == "" {
		cfg1.IndexPrefix = "coco_"
	}
	esClient := elastic.GetClient(global.MustLookupString(elastic.GlobalSystemElasticsearchID))
	var docType = "_doc"
	version := esClient.GetVersion()
	if v := esClient.GetMajorVersion(); v > 0 && v < 7 && version.Distribution == elastic.Elasticsearch {
		docType = "doc"
	}
	output := tpl.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		switch tag {
		case "SETUP_INDEX_PREFIX":
			return w.Write([]byte(cfg1.IndexPrefix))
		case "SETUP_DOC_TYPE":
			return w.Write([]byte(docType))
		}
		//ignore unresolved variable
		return w.Write([]byte("$[[" + tag + "]]"))
	})
	br := bytes.NewReader([]byte(output))
	scanner := bufio.NewScanner(br)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)
	scanner.Split(bufio.ScanLines)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	var setupHTTPPool = fasthttp.NewRequestResponsePool("setup")
	req := setupHTTPPool.AcquireRequest()
	res := setupHTTPPool.AcquireResponse()

	defer setupHTTPPool.ReleaseRequest(req)
	defer setupHTTPPool.ReleaseResponse(res)
	esConfig := elastic.GetConfig(global.MustLookupString(elastic.GlobalSystemElasticsearchID))
	var endpoint = esConfig.Endpoint
	if endpoint == "" && len(esConfig.Endpoints) > 0 {
		endpoint = esConfig.Endpoints[0]
	}
	parts := strings.Split(endpoint, "://")
	if len(parts) != 2 {
		return fmt.Errorf("invalid elasticsearch endpoint [%s]", endpoint)
	}
	var (
		username = ""
		password = ""
	)
	if esConfig.BasicAuth != nil {
		username = esConfig.BasicAuth.Username
		password = esConfig.BasicAuth.Password.Get()
	}

	_, err, _ = replay.ReplayLines(req, res, pipeline.AcquireContext(pipeline.PipelineConfigV2{}), lines, parts[0], parts[1], username, password)
	return err
}

func (h *APIHandler) initializeSetupTemplates(userID string, setupCfg SetupConfig, serverEndpoint string) error {
	if setupCfg.Language != "en-US" {
		setupCfg.Language = "zh-CN"
	}
	baseDir := path.Join(global.Env().GetConfigDir(), "setup", setupCfg.Language)
	cfg1 := elastic1.ORMConfig{}
	exist, err := env.ParseConfig("elastic.orm", &cfg1)
	if exist && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}

	if cfg1.IndexPrefix == "" {
		cfg1.IndexPrefix = "coco_"
	}
	esClient := elastic.GetClient(global.MustLookupString(elastic.GlobalSystemElasticsearchID))
	var docType = "_doc"
	version := esClient.GetVersion()
	if v := esClient.GetMajorVersion(); v > 0 && v < 7 && version.Distribution == elastic.Elasticsearch {
		docType = "doc"
	}
	return filepath.Walk(baseDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %v", path, err)
		}
		if info.IsDir() {
			return nil
		}
		// skip file which is not template file
		if !strings.HasSuffix(path, ".tpl") {
			return nil
		}
		return h.initializeTemplate(userID, path, cfg1.IndexPrefix, docType, &setupCfg, serverEndpoint)
	})
}

func (h *APIHandler) initializeTemplate(userID string, dslTplFile string, indexPrefix string, docType string, setupCfg *SetupConfig, serverEndpoint string) error {
	dsl, err := util.FileGetContent(dslTplFile)
	if err != nil {
		return err
	}
	if len(dsl) == 0 {
		return fmt.Errorf("got empty template [%s]", dslTplFile)
	}

	var tpl *fasttemplate.Template
	tpl, err = fasttemplate.NewTemplate(string(dsl), "$[[", "]]")

	if tpl == nil {
		panic("invalid template file")
	}

	output := tpl.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		switch tag {
		case "SETUP_OWNER_ID":
			return w.Write([]byte(userID))
		case "SETUP_INDEX_PREFIX":
			return w.Write([]byte(indexPrefix))
		case "SETUP_SCHEMA_VER":
			return w.Write([]byte(common.GetSchemaSuffix()))
		case "SETUP_DOC_TYPE":
			return w.Write([]byte(docType))
		case "SETUP_SERVER_ENDPOINT":
			return w.Write([]byte(serverEndpoint))
		}
		//ignore unresolved variable
		return w.Write([]byte("$[[" + tag + "]]"))
	})
	br := bytes.NewReader([]byte(output))
	scanner := bufio.NewScanner(br)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)
	scanner.Split(bufio.ScanLines)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	var setupHTTPPool = fasthttp.NewRequestResponsePool("setup")
	req := setupHTTPPool.AcquireRequest()
	res := setupHTTPPool.AcquireResponse()

	defer setupHTTPPool.ReleaseRequest(req)
	defer setupHTTPPool.ReleaseResponse(res)
	esConfig := elastic.GetConfig(global.MustLookupString(elastic.GlobalSystemElasticsearchID))
	var endpoint = esConfig.Endpoint
	if endpoint == "" && len(esConfig.Endpoints) > 0 {
		endpoint = esConfig.Endpoints[0]
	}
	parts := strings.Split(endpoint, "://")
	if len(parts) != 2 {
		return fmt.Errorf("invalid elasticsearch endpoint [%s]", endpoint)
	}
	var (
		username = ""
		password = ""
	)
	if esConfig.BasicAuth != nil {
		username = esConfig.BasicAuth.Username
		password = esConfig.BasicAuth.Password.Get()
	}

	_, err, _ = replay.ReplayLines(req, res, pipeline.AcquireContext(pipeline.PipelineConfigV2{}), lines, parts[0], parts[1], username, password)
	return err
}

// applySetupDefaultModels processes the language/vision/embedding selections
// from the setup wizard. For each provided selection it:
//  1. Updates an existing builtin provider (when ModelProvider.ID is set) with
//     the user's API token, or
//  2. Creates a new custom provider on the fly (when ModelProvider.ID is empty
//     but other fields describe a provider), and
//  3. Records a ModelId reference in info.DefaultModel so settings persistence
//     picks them up.
//
// Skipped silently when input has no model selections at all.
func (h *APIHandler) applySetupDefaultModels(req *http.Request, input *SetupDefaultModelConfig, info *core.Config) error {
	if input.LanguageModel == nil && input.VisionModel == nil && input.EmbeddingModel == nil {
		return nil
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh

	resolve := func(m *SetupModelConfig, llmType core.LLMType) (*core.ModelId, error) {
		if m == nil {
			return nil, nil
		}
		providerID, modelName, err := ensureSetupModelProvider(ctx, m, llmType)
		if err != nil {
			return nil, err
		}
		return &core.ModelId{ProviderID: providerID, ID: modelName}, nil
	}

	languageRef, err := resolve(input.LanguageModel, core.LLMTypeLanguage)
	if err != nil {
		return fmt.Errorf("apply language model: %w", err)
	}
	visionRef, err := resolve(input.VisionModel, core.LLMTypeVision)
	if err != nil {
		return fmt.Errorf("apply vision model: %w", err)
	}
	embeddingRef, err := resolve(input.EmbeddingModel, core.LLMTypeEmbedding)
	if err != nil {
		return fmt.Errorf("apply embedding model: %w", err)
	}

	if info.DefaultModel == nil {
		info.DefaultModel = &core.DefaultModel{}
	}
	if languageRef != nil {
		info.DefaultModel.LanguageModel = languageRef
	}
	if visionRef != nil {
		info.DefaultModel.VisionModel = visionRef
	}
	if embeddingRef != nil {
		info.DefaultModel.EmbeddingModel = embeddingRef
	}
	return nil
}

// validateDefaultModelConfig validates the default-model setup payload. Rules:
//  1. Exactly one of model_id or model must be set per selection.
//  2. When model is set, model.id is required.
//  3. support_reasoning is only valid on language model selections; it is
//     rejected for vision and embedding models.
//  4. A custom provider (ModelProvider.ID is empty) must use the model field
//     to define a new model — model_id is not allowed because there are no
//     pre-registered models to reference on a provider that does not exist yet.
//  5. When multiple selections reference the same builtin provider (matched by
//     ModelProvider.ID), they must all carry the same api_token.
func validateDefaultModelConfig(input *SetupDefaultModelConfig) error {
	type roleEntry struct {
		name   string
		cfg    *SetupModelConfig
		isLang bool
	}
	roles := []roleEntry{
		{"language_model", input.LanguageModel, true},
		{"vision_model", input.VisionModel, false},
		{"embedding_model", input.EmbeddingModel, false},
	}

	type seenProvider struct {
		role  string
		token string
	}
	seenProviders := map[string]seenProvider{}

	for _, r := range roles {
		if r.cfg == nil {
			continue
		}
		hasID := r.cfg.ModelID != ""
		hasModel := r.cfg.Model != nil

		// Rule 1
		if hasID == hasModel {
			return fmt.Errorf("%s: exactly one of model_id or model must be set", r.name)
		}
		if hasModel {
			// Rule 2
			if r.cfg.Model.ID == "" {
				return fmt.Errorf("%s: model.id is required", r.name)
			}
			// Rule 3
			if !r.isLang && r.cfg.Model.SupportReasoning {
				return fmt.Errorf("%s: support_reasoning is only valid for language models", r.name)
			}
		}
		// Rule 4
		if r.cfg.ModelProvider.ID == "" && hasID {
			return fmt.Errorf("%s: model_id cannot be used with a custom provider; use model instead", r.name)
		}
		// Rule 5
		if pid := r.cfg.ModelProvider.ID; pid != "" {
			if prev, ok := seenProviders[pid]; ok {
				if prev.token != r.cfg.APIToken {
					return fmt.Errorf(
						"conflicting api_token for the same model provider used by %s and %s",
						prev.role, r.name,
					)
				}
			} else {
				seenProviders[pid] = seenProvider{role: r.name, token: r.cfg.APIToken}
			}
		}
	}
	return nil
}

// ensureSetupModelProvider creates or updates the underlying model provider for
// a single setup-wizard model selection and returns (providerID, modelName).
// llmType identifies the role the selection plays so that any newly added
// model carries the correct Type when written to ES.
func ensureSetupModelProvider(ctx *orm.Context, m *SetupModelConfig, llmType core.LLMType) (string, string, error) {
	sp := m.ModelProvider

	modelName := m.ModelID
	if m.Model != nil {
		modelName = m.Model.ID
	}

	// Existing provider (builtin or previously created): update API key,
	// optionally appending a freshly defined model.
	if sp.ID != "" {
		provider := core.ModelProvider{}
		provider.ID = sp.ID
		exists, err := orm.GetV2(ctx, &provider)
		if err != nil {
			return "", "", err
		}
		if !exists {
			return "", "", fmt.Errorf("model provider [%s] not found", sp.ID)
		}
		provider.APIKey = m.APIToken
		provider.Enabled = true
		if m.Model != nil {
			upsertProviderModel(&provider, *m.Model, llmType)
		}
		if err := orm.Update(ctx, &provider); err != nil {
			return "", "", err
		}
		common.GeneralObjectCache.Delete(common.ModelProviderCachePrimary, provider.ID)
		return provider.ID, modelName, nil
	}

	// Custom provider: create a new one with the selected model registered.
	provider := &core.ModelProvider{
		Name:        sp.Name,
		Description: sp.Description,
		Icon:        sp.Icon,
		BaseURL:     sp.BaseURL,
		APIType:     sp.APIType,
		APIKey:      m.APIToken,
		Enabled:     true,
		Builtin:     false,
	}
	provider.Models = []core.ModelConfig{newProviderModel(modelName, m.Model, llmType)}
	if err := orm.Create(ctx, provider); err != nil {
		return "", "", err
	}
	return provider.ID, modelName, nil
}

// newProviderModel builds a ModelConfig for an ES write, carrying the role's
// LLM type and (when provided) the SupportReasoning flag.
func newProviderModel(name string, def *SetupModelDef, llmType core.LLMType) core.ModelConfig {
	mc := core.ModelConfig{Name: name, Type: llmType}
	if def != nil {
		mc.SupportReasoning = def.SupportReasoning
	}
	return mc
}

// upsertProviderModel adds the freshly-defined model to provider.Models, or
// updates the matching entry in place when one with the same name already
// exists.
func upsertProviderModel(provider *core.ModelProvider, def SetupModelDef, llmType core.LLMType) {
	for i := range provider.Models {
		if provider.Models[i].Name == def.ID {
			provider.Models[i].Type = llmType
			provider.Models[i].SupportReasoning = def.SupportReasoning
			return
		}
	}
	provider.Models = append(provider.Models, newProviderModel(def.ID, &def, llmType))
}
