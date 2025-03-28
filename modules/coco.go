/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package modules

import (
	"errors"
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant"
	_ "infini.sh/coco/modules/assistant"
	"infini.sh/coco/modules/common"
	_ "infini.sh/coco/modules/connector"
	_ "infini.sh/coco/modules/indexing"
	"infini.sh/coco/modules/integration"
	_ "infini.sh/coco/modules/integration"
	_ "infini.sh/coco/modules/search"
	_ "infini.sh/coco/modules/system"
	cfg "infini.sh/framework/core/api/common"
	"infini.sh/framework/core/api/websocket"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"net/http"
	"time"
)

type Coco struct {
}

func (this *Coco) Setup() {
	orm.MustRegisterSchemaWithIndexName(assistant.Session{}, "session")
	orm.MustRegisterSchemaWithIndexName(common.Document{}, "document")
	orm.MustRegisterSchemaWithIndexName(assistant.ChatMessage{}, "message")
	orm.MustRegisterSchemaWithIndexName(common.Attachment{}, "attachment")
	orm.MustRegisterSchemaWithIndexName(common.Connector{}, "connector")
	orm.MustRegisterSchemaWithIndexName(common.DataSource{}, "datasource")
	orm.MustRegisterSchemaWithIndexName(common.Integration{}, "integration")

	cocoConfig := common.Config{
		LLMConfig: &common.LLMConfig{
			Type:                "deepseek",
			DefaultModel:        "deepseek-r1",
			IntentAnalysisModel: "tongyi-intent-detect-v3",
			PickingDocModel:     "deepseek-r1-distill-qwen-32b",
			AnsweringModel:      "deepseek-r1",
			ContextLength:       131072,
			Keepalive:           "30m",
			Endpoint:            "https://dashscope.aliyuncs.com/compatible-mode/v1",
		},
		ServerInfo: &common.ServerInfo{Version: common.Version{Number: global.Env().GetVersion()}, Updated: time.Now()},
	}

	ok, err := env.ParseConfig("coco", &cocoConfig)
	if ok && err != nil {
		panic(err)
	}

	//update coco's config
	global.Register("APP_CONFIG", &cocoConfig)

	websocket.RegisterConnectCallback(func(sessionID string, w http.ResponseWriter, r *http.Request) error {
		log.Trace("websocket established: ", sessionID)
		if cfg.IsAuthEnable() {
			claims, err := core.ValidateLogin(r)
			if err != nil {
				return err
			}
			if claims != nil {

				//log.Info(claims.Provider)
				//log.Info(claims.Login)  //external login within provider
				//log.Info(claims.UserId) //internal system user's id
				//log.Info(claims.Roles)

				if claims.UserId != "" {

					err := kv.AddValue(common.WEBSOCKET_USER_SESSION, []byte(claims.UserId), []byte(sessionID))
					if err != nil {
						log.Error(err)
					}
					err = kv.AddValue(common.WEBSOCKET_SESSION_USER, []byte(sessionID), []byte(claims.UserId))
					if err != nil {
						log.Error(err)
					}

					log.Debugf("established websocket: %v for user: %v", sessionID, claims.UserId)

				} else {
					return errors.New("invalid claims")
				}
			}
		}
		return nil
	})

	websocket.RegisterDisconnectCallback(func(sessionID string) {
		v, err := kv.GetValue(common.WEBSOCKET_SESSION_USER, []byte(sessionID))
		if err != nil {
			log.Error(err)
			return
		}

		if v != nil && len(v) > 0 {
			err := kv.DeleteKey(common.WEBSOCKET_USER_SESSION, v)
			if err != nil {
				log.Error(err)
			}

			err = kv.DeleteKey(common.WEBSOCKET_SESSION_USER, []byte(sessionID))
			if err != nil {
				log.Error(err)
			}
		}

		log.Debug("websocket disconnected: ", sessionID)
	})

}

func (this *Coco) Start() error {
	integration.InitIntegrationOrigins()
	return nil
}

func (this *Coco) Stop() error {
	return nil
}

func (this *Coco) Name() string {
	return "coco"
}
