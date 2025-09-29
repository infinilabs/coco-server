package sharepoint

import (  
	"context"  
	"fmt"  
	"net/http"  
  
	"github.com/julienschmidt/httprouter"  
	log "github.com/cihub/seelog"  
	"golang.org/x/oauth2"  
	"golang.org/x/oauth2/microsoft"  
	"infini.sh/coco/modules/common"  
	"infini.sh/framework/core/api"  
	"infini.sh/framework/core/orm"  
) 

func (p *Plugin) connect(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	datasourceID := req.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		api.WriteError(w, fmt.Errorf("datasource_id is required"), http.StatusBadRequest)
		return
	}

	// 获取数据源配置
	datasource := &common.DataSource{}
	datasource.ID = datasourceID
	exists, err := orm.Get(datasource)
	if !exists || err != nil {
		api.WriteError(w, fmt.Errorf("datasource not found"), http.StatusNotFound)
		return
	}

	config, err := parseSharePointConfig(datasource)
	if err != nil {
		api.WriteError(w, err, http.StatusBadRequest)
		return
	}

	// 创建OAuth配置
	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint:     microsoft.AzureADEndpoint(config.TenantID),
		Scopes:       []string{"https://graph.microsoft.com/.default"},
		RedirectURL: fmt.Sprintf("%s/connector/sharepoint/oauth_redirect?datasource_id=%s",
			getBaseURL(req), datasourceID),
	}

	// 生成授权URL
	authURL := oauthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)

	api.WriteJSON(w, map[string]interface{}{
		"auth_url": authURL,
	}, http.StatusOK)
}

func (p *Plugin) oAuthRedirect(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	datasourceID := req.URL.Query().Get("datasource_id")
	code := req.URL.Query().Get("code")

	if datasourceID == "" || code == "" {
		api.WriteError(w, fmt.Errorf("missing required parameters"), http.StatusBadRequest)
		return
	}

	// 获取数据源
	datasource := &common.DataSource{}
	datasource.ID = datasourceID
	exists, err := orm.Get(datasource)
	if !exists || err != nil {
		api.WriteError(w, fmt.Errorf("datasource not found"), http.StatusNotFound)
		return
	}

	config, err := parseSharePointConfig(datasource)
	if err != nil {
		api.WriteError(w, err, http.StatusBadRequest)
		return
	}

	// 交换token
	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint:     microsoft.AzureADEndpoint(config.TenantID),
		Scopes:       []string{"https://graph.microsoft.com/.default"},
		RedirectURL: fmt.Sprintf("%s/connector/sharepoint/oauth_redirect?datasource_id=%s",
			getBaseURL(req), datasourceID),
	}

	ctx := context.Background()
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		api.WriteError(w, fmt.Errorf("failed to exchange token: %w", err), http.StatusInternalServerError)
		return
	}

	// 更新数据源配置
	configMap := datasource.Connector.Config.(map[string]interface{})
	configMap["access_token"] = token.AccessToken
	configMap["refresh_token"] = token.RefreshToken
	configMap["token_expiry"] = token.Expiry

	datasource.Connector.Config = configMap
	err = orm.Update(datasource)
	if err != nil {
		api.WriteError(w, fmt.Errorf("failed to update datasource: %w", err), http.StatusInternalServerError)
		return
	}

	// 重定向到成功页面
	http.Redirect(w, req, "/datasource/edit/"+datasourceID+"?connected=true", http.StatusFound)
}

func (p *Plugin) reset(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	datasourceID := req.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		api.WriteError(w, fmt.Errorf("datasource_id is required"), http.StatusBadRequest)
		return
	}

	// 获取数据源
	datasource := &common.DataSource{}
	datasource.ID = datasourceID
	exists, err := orm.Get(datasource)
	if !exists || err != nil {
		api.WriteError(w, fmt.Errorf("datasource not found"), http.StatusNotFound)
		return
	}

	// 清除token
	configMap := datasource.Connector.Config.(map[string]interface{})
	delete(configMap, "access_token")
	delete(configMap, "refresh_token")
	delete(configMap, "token_expiry")

	datasource.Connector.Config = configMap
	err = orm.Update(datasource)
	if err != nil {
		api.WriteError(w, fmt.Errorf("failed to update datasource: %w", err), http.StatusInternalServerError)
		return
	}

	api.WriteJSON(w, map[string]interface{}{
		"success": true,
	}, http.StatusOK)
}

func getBaseURL(req *http.Request) string {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, req.Host)
}
