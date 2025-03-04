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
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/security"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
	"net/http"
	"time"
)

type SetupConfig struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	LLM      struct {
		Type         string `json:"type,omitempty"`
		Endpoint     string `json:"endpoint,omitempty"`
		DefaultModel string `json:"default_model,omitempty"`
	} `json:"llm,omitempty"`
}

var SetupLock = ".setup_lock"

func checkSetupStatus() bool {
	exists, err := kv.ExistsKey(core.DefaultSettingBucketKey, []byte(SetupLock))
	if exists || err != nil {
		global.Env().EnableSetup(false)
		return true
	}
	global.Env().EnableSetup(true)
	return false
}

func (h *APIHandler) setupServer(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	isSetup := checkSetupStatus()
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
		info.ServerInfo.Name = input.Name
	} else if info.ServerInfo.Name == "" {
		info.ServerInfo.Name = "My Coco Server"
	}

	if input.Password == "" {
		panic("password can't be empty")
	}

	//save user's profile
	profile := core.User{Name: input.Name}
	profile.ID = "default_user_id"
	profile.Email = input.Email

	err = kv.AddValue(core.DefaultSettingBucketKey, []byte(core.DefaultUserProfileKey), util.MustToJSONBytes(profile))
	if err != nil {
		panic(err)
	}
	//save user's password
	err = security.SavePassword(input.Password)
	if err != nil {
		panic(err)
	}
	//save server's config
	err = kv.AddValue(core.DefaultSettingBucketKey, []byte(core.DefaultServerConfigKey), util.MustToJSONBytes(info.ServerInfo))
	if err != nil {
		panic(err)
	}

	//setup lock
	err = kv.AddValue(core.DefaultSettingBucketKey, []byte(SetupLock), []byte(time.Now().String()))
	if err != nil {
		panic(err)
	}

	h.WriteAckOKJSON(w)
}

func clearSetupLock() {
	err := kv.DeleteKey(core.DefaultSettingBucketKey, []byte(SetupLock))
	if err != nil {
		panic(err)
	}
}
