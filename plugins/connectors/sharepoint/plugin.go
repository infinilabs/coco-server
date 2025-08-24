/* Copyright © INFINI LTD. All rights reserved.  
 * Web: https://infinilabs.com  
 * Email: hello#infini.ltd */  
  
 package sharepoint  
  
 import (  
	"context"  
	"fmt"  
	"time"  
  
	log "github.com/cihub/seelog"  
	"github.com/julienschmidt/httprouter"  
	"golang.org/x/oauth2/microsoft"  
	"infini.sh/coco/modules/common"  
	"infini.sh/coco/plugins/connectors"  
	"infini.sh/framework/core/api"  
	"infini.sh/framework/core/global"  
	"infini.sh/framework/core/module"  
	"infini.sh/framework/core/orm"  
	"infini.sh/framework/core/queue"  
	"infini.sh/framework/core/task"  
	"infini.sh/framework/core/util"  
)  
   
 const ConnectorSharePoint = "sharepoint"  
   
 type Plugin struct {  
	 connectors.BasePlugin  
	 apiClient *SharePointAPIClient  
 }  
   
 func init() {  
	 module.RegisterUserPlugin(&Plugin{})  
 }  
   
 func (p *Plugin) Setup() {  
	 p.BasePlugin.Init(fmt.Sprintf("connector.%s", ConnectorSharePoint), "indexing sharepoint", p)  
	   
	 // 注册API端点  
	 api.HandleUIMethod(api.GET, "/connector/sharepoint/connect", p.connect, api.RequireLogin())  
	 api.HandleUIMethod(api.POST, "/connector/sharepoint/reset", p.reset, api.RequireLogin())  
	 api.HandleUIMethod(api.GET, "/connector/sharepoint/oauth_redirect", p.oAuthRedirect, api.RequireLogin())  
 }  
   
 func (p *Plugin) Start() error {  
	 return p.BasePlugin.Start(time.Second * 30)  
 }  
   
 func (p *Plugin) Stop() error {  
	 return nil  
 }  
   
 func (p *Plugin) Name() string {  
	 return ConnectorSharePoint  
 }  
   
 func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {  
	 log.Infof("[sharepoint connector] starting scan for datasource [%s]", datasource.Name)  
	   
	 config, err := parseSharePointConfig(datasource)  
	 if err != nil {  
		 log.Errorf("[sharepoint connector] failed to parse config: %v", err)  
		 return  
	 }  
	   
	 client, err := NewSharePointAPIClient(config)  
	 if err != nil {  
		 log.Errorf("[sharepoint connector] failed to create API client: %v", err)  
		 return  
	 }  
	   
	 p.apiClient = client  
	   
	 // 开始同步过程  
	 err = p.syncSharePointContent(connector, datasource, config)  
	 if err != nil {  
		 log.Errorf("[sharepoint connector] sync failed: %v", err)  
		 return  
	 }  
	   
	 log.Infof("[sharepoint connector] completed scan for datasource [%s]", datasource.Name)  
 }