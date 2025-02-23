package oauth

import (
	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
	"net/http"
	"time"
)

func (h *APIHandler) requestToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	token := req.Header.Get("X-API-TOKEN")
	if token == "" {
		token= h.GetParameter(req, "token")
		if token==""{
			panic("invalid token")
		}
	}

	log.Trace("request access token, old token: ", token)

	//TODO valid token and user's information

	//username := h.GetCurrentUser(req)
	//
	//log.Error("username:", username)

	reqID := h.MustGetParameter(w, req, "request_id")

	item := h.cCache.GetOrCreateSecondaryCache("sso_temp_token").Get(reqID)
	if item != nil && !item.Expired() {
		payload, ok := item.Value().(util.MapStr)
		if ok {
			if payload["request_id"].(string) == reqID && payload["code"].(string) == token {
				//valid user with valid code

				//TODO
				provider := payload["provider"].(string)
				username := payload["login"].(string)
				userid := payload["userid"].(string)
				//if payload["username"].(string) != username {
				//	log.Error(payload["username"], ",", username)
				//	panic("invalid user")
				//}

				res := util.MapStr{}
				accessToken := util.GetUUID() + util.GenerateRandomString(64)
				res["access_token"] = accessToken
				expiredAT := time.Now().Add(365 * 24 * time.Hour).Unix()
				res["expire_at"] = expiredAT

				newPayload := util.MapStr{}
				newPayload["provider"] = provider
				newPayload["login"] = username
				newPayload["userid"] = userid
				newPayload["expire_at"] = expiredAT

				if username == "" || userid == "" {
					panic("invalid user info")
				}

				log.Trace("save:", util.MustToJSON(newPayload))

				//TODO save access token to store
				err := kv.AddValue("access_token", []byte(accessToken), util.MustToJSONBytes(newPayload))
				if err != nil {
					panic(err)
				}
				h.WriteJSON(w, res, 200)

				return
			} else {
				log.Debug("invalid temp token, mismatched, ", reqID, ",", token)
			}
		} else {
			log.Debug("invalid temp token, not found, ", reqID, ",", token)
		}

	} else {
		log.Debug("invalid temp token, or expired, ", reqID, ",", token)
	}

	h.WriteError(w, "invalid request", 400)

}
