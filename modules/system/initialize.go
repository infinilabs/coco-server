// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

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

	security2 "infini.sh/framework/core/security"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/security"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
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
	LLM      struct {
		Type         string `json:"type,omitempty"`
		Endpoint     string `json:"endpoint,omitempty"`
		DefaultModel string `json:"default_model,omitempty"`
		Token        string `json:"token,omitempty"`
		Reasoning    bool   `json:"reasoning,omitempty"` // Whether to enable reasoning mode
	} `json:"llm,omitempty"`
	Language string `json:"language,omitempty"`
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

	//save user's profile
	profile := security2.UserProfile{Name: input.Name}
	profile.Email = input.Email
	if info.ServerInfo.Managed {
		profile.ID = core.DefaultUserLogin
		err = kv.AddValue(core.UserProfileKey, []byte(profile.ID), util.MustToJSONBytes(profile))
	} else {
		//keep backward compatibility, TODO to be removed
		profile.ID = core.DefaultUserLogin
		err = kv.AddValue(core.DefaultSettingBucketKey, []byte(core.DefaultUserProfileKey), util.MustToJSONBytes(profile))
	}

	if err != nil {
		panic(err)
	}
	//save user's password
	err = security.SavePassword(input.Password)
	if err != nil {
		panic(err)
	}
	//initialize setup templates
	err = h.initializeSetupTemplates(input, info.ServerInfo.Endpoint)
	if err != nil {
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

func (h *APIHandler) initializeSetupTemplates(setupCfg SetupConfig, serverEndpoint string) error {
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
			if err != nil {
				return fmt.Errorf("error accessing path %s: %v", path, err)
			}
		}
		if info.IsDir() {
			return nil
		}
		// skip file which is not template file
		if !strings.HasSuffix(path, ".tpl") {
			return nil
		}
		return h.initializeTemplate(path, cfg1.IndexPrefix, docType, &setupCfg, serverEndpoint)
	})
}

func (h *APIHandler) initializeTemplate(dslTplFile string, indexPrefix string, docType string, setupCfg *SetupConfig, serverEndpoint string) error {
	dsl, err := util.FileGetContent(dslTplFile)
	if err != nil {
		return err
	}
	if len(dsl) == 0 {
		return fmt.Errorf("got empty template [%s]", dslTplFile)
	}

	var tpl *fasttemplate.Template
	tpl, err = fasttemplate.NewTemplate(string(dsl), "$[[", "]]")
	var (
		modelProvideEnabled = false
		apiKey              = ""
		apiType             = "openai"
		baseURL             = ""
		defaultModel        = ""
		answeringModel      = "null"
	)
	if setupCfg.LLM.Endpoint != "" {
		modelProvideEnabled = true
		apiKey = setupCfg.LLM.Token
		apiType = setupCfg.LLM.Type
		baseURL = setupCfg.LLM.Endpoint
		defaultModel = fmt.Sprintf(`{"name": "%s"}`, setupCfg.LLM.DefaultModel)
		answeringModel = fmt.Sprintf(`{"provider_id": "coco", "name": "%s",  "prompt": { "template": "{{.query}}" }}`, setupCfg.LLM.DefaultModel)
	}

	output := tpl.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		switch tag {
		case "SETUP_INDEX_PREFIX":
			return w.Write([]byte(indexPrefix))
		case "SETUP_SCHEMA_VER":
			return w.Write([]byte(common.GetSchemaSuffix()))
		case "SETUP_DOC_TYPE":
			return w.Write([]byte(docType))
		case "SETUP_LLM_ENABLED":
			return w.Write([]byte(fmt.Sprintf("%v", modelProvideEnabled)))
		case "SETUP_LLM_API_KEY":
			return w.Write([]byte(apiKey))
		case "SETUP_LLM_API_TYPE":
			return w.Write([]byte(apiType))
		case "SETUP_LLM_BASE_URL":
			return w.Write([]byte(baseURL))
		case "SETUP_LLM_DEFAULT_MODEL":
			return w.Write([]byte(defaultModel))
		case "SETUP_LLM_REASONING":
			return w.Write([]byte(fmt.Sprintf("%v", setupCfg.LLM.Reasoning)))
		case "SETUP_ASSISTANT_ANSWERING_MODEL":
			return w.Write([]byte(answeringModel))
		case "SETUP_LLM_PROVIDER_ID":
			return w.Write([]byte("coco"))
		case "SETUP_LLM_DEFAULT_MODEL_ID":
			return w.Write([]byte(setupCfg.LLM.DefaultModel))
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
