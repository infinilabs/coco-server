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
	// Default models picked in the setup wizard. All fields are optional;
	// the wizard allows users to skip model configuration entirely.
	LanguageModel  *SetupModelConfig `json:"language_model,omitempty"`
	VisionModel    *SetupModelConfig `json:"vision_model,omitempty"`
	EmbeddingModel *SetupModelConfig `json:"embedding_model,omitempty"`
	Language       string            `json:"language,omitempty"`
}

// SetupModelConfig describes a single model selection in the setup wizard.
//
// Two shapes are accepted:
//  1. Built-in provider: ModelProvider.ID identifies an existing builtin provider,
//     and APIToken / ModelID are supplied by the user.
//  2. Custom provider: ModelProvider carries the full provider definition
//     (Name, BaseURL, APIType, etc.) to be created on the fly.
type SetupModelConfig struct {
	ModelProvider SetupModelProvider `json:"model_provider,omitempty"`
	ModelID       string             `json:"model_id,omitempty"`
	APIToken      string             `json:"api_token,omitempty"`
}

// SetupModelProvider is either a reference (ID only) to a builtin provider,
// or a full custom provider definition.
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
	APIToken    string `json:"api_token,omitempty"`
}

var SetupLock = ".setup_lock"

func isAlreadyDoneSetup() bool {
	exists, err := kv.ExistsKey(core.DefaultSettingBucketKey, []byte(SetupLock))
	if exists || err != nil {
		global.Env().EnableSetup(false)
		return true
	}
	global.Env().EnableSetup(true)
	return false
}

func (h *APIHandler) setupServer(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	isSetup := isAlreadyDoneSetup()
	if isSetup {
		panic("the server has already been initialized")
	}

	input := SetupConfig{}
	err := h.DecodeJSON(req, &input)
	if err != nil {
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

		//initialize setup templates
		err = h.initializeSetupTemplates(user.ID, input, info.ServerInfo.Endpoint)
		if err != nil {
			panic(err)
		}
	}

	// Apply default-model selections from the wizard: create/update the underlying
	// model providers as needed, then record references in info.DefaultModel.
	if err := h.applySetupDefaultModels(req, &input, &info); err != nil {
		panic(err)
	}

	//setup lock
	err = kv.AddValue(core.DefaultSettingBucketKey, []byte(SetupLock), []byte(time.Now().String()))
	if err != nil {
		panic(err)
	}
	//save app config
	common.SetAppConfig(&info)

	h.WriteAckOKJSON(w)
}

func clearSetupLock() {
	err := kv.DeleteKey(core.DefaultSettingBucketKey, []byte(SetupLock))
	if err != nil {
		panic(err)
	}
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
func (h *APIHandler) applySetupDefaultModels(req *http.Request, input *SetupConfig, info *core.Config) error {
	if input.LanguageModel == nil && input.VisionModel == nil && input.EmbeddingModel == nil {
		return nil
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh

	resolve := func(m *SetupModelConfig) (*core.ModelId, error) {
		if m == nil {
			return nil, nil
		}
		providerID, err := ensureSetupModelProvider(ctx, m)
		if err != nil {
			return nil, err
		}
		return &core.ModelId{ProviderID: providerID, ID: m.ModelID}, nil
	}

	languageRef, err := resolve(input.LanguageModel)
	if err != nil {
		return fmt.Errorf("apply language model: %w", err)
	}
	visionRef, err := resolve(input.VisionModel)
	if err != nil {
		return fmt.Errorf("apply vision model: %w", err)
	}
	embeddingRef, err := resolve(input.EmbeddingModel)
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

// ensureSetupModelProvider creates or updates the underlying model provider for
// a single setup-wizard model selection and returns its provider ID.
func ensureSetupModelProvider(ctx *orm.Context, m *SetupModelConfig) (string, error) {
	sp := m.ModelProvider

	// Built-in provider: update its API key in place.
	if sp.ID != "" {
		provider := core.ModelProvider{}
		provider.ID = sp.ID
		exists, err := orm.GetV2(ctx, &provider)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("model provider [%s] not found", sp.ID)
		}
		provider.APIKey = m.APIToken
		provider.Enabled = true
		if err := orm.Update(ctx, &provider); err != nil {
			return "", err
		}
		common.GeneralObjectCache.Delete(common.ModelProviderCachePrimary, provider.ID)
		return provider.ID, nil
	}

	// Custom provider: create a new one.
	provider := &core.ModelProvider{
		Name:        sp.Name,
		Description: sp.Description,
		Icon:        sp.Icon,
		BaseURL:     sp.BaseURL,
		APIType:     sp.APIType,
		APIKey:      sp.APIToken,
		Enabled:     true,
		Builtin:     false,
	}
	if m.ModelID != "" {
		provider.Models = []core.ModelConfig{{Name: m.ModelID}}
	}
	if err := orm.Create(ctx, provider); err != nil {
		return "", err
	}
	return provider.ID, nil
}
