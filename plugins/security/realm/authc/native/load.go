// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Console is offered under the GNU Affero General Public License v3.0
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

/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package native

import (
	_ "embed"
	"github.com/mitchellh/mapstructure"
	"infini.sh/coco/plugins/security/core"
	"infini.sh/framework/core/elastic"
	"path"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/segmentio/encoding/json"
	"infini.sh/framework/core/api/routetree"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

type ElasticsearchAPIMetadata struct {
	Name    string   `json:"name"`
	Methods []string `json:"methods"`
	Path    string   `json:"path"`
}

type ElasticsearchAPIMetadataList []ElasticsearchAPIMetadata

func (list ElasticsearchAPIMetadataList) GetNames() []string {
	var names []string
	for _, md := range list {
		if !util.StringInArray(names, md.Name) {
			names = append(names, md.Name)
		}
	}
	return names
}

//go:embed permission.json
var permissionFile []byte

func loadJsonConfig() map[string]ElasticsearchAPIMetadataList {
	externalConfig := path.Join(global.Env().GetConfigDir(), "permission.json")
	if util.FileExists(externalConfig) {
		log.Infof("loading permission file from %s", externalConfig)
		bytes, err := util.FileGetContent(externalConfig)
		if err != nil {
			log.Errorf("load permission file failed, use embedded config, err: %v", err)
		} else {
			permissionFile = bytes
		}
	}
	apis := map[string]ElasticsearchAPIMetadataList{}
	err := json.Unmarshal(permissionFile, &apis)
	if err != nil {
		log.Error("json config unmarshal err " + err.Error())
		return nil
	}

	return apis
}

func Init() {

	//load local files
	apis := loadJsonConfig()
	if apis != nil {
		var esAPIRouter = routetree.New()
		for _, list := range apis {
			for _, md := range list {
				//skip wildcard *
				if strings.HasSuffix(md.Path, "*") {
					continue
				}
				for _, method := range md.Methods {
					esAPIRouter.Handle(method, md.Path, md.Name)
				}
			}
		}
		core.RegisterAPIPermissionRouter("elasticsearch", esAPIRouter)
	}

	permissions := map[string]interface{}{
		"index_privileges": apis["indices"].GetNames(),
	}
	delete(apis, "indices")
	otherApis := map[string][]string{}
	for key, list := range apis {
		otherApis[key] = list.GetNames()

	}
	permissions["cluster_privileges"] = otherApis
	core.RegisterPermission(core.Elasticsearch, permissions)

	//load role from store
	loadRemoteRolePermission()
}

func loadRemoteRolePermission() {
	log.Trace("start loading roles from adapter")
	core.RoleMap = make(map[string]core.Role)
	for k, role := range core.BuiltinRoles {
		core.RoleMap[k] = role
	}

	log.Debug("load security permissions,", core.RoleMap, core.BuiltinRoles)

	res, err := handler.Role.Search("", 0, 1000)
	if err != nil {
		log.Error(err)
		return
	}
	response := elastic.SearchResponse{}
	util.FromJSONBytes(res.Raw, &response)

	for _, v := range response.Hits.Hits {
		var role core.Role
		delete(v.Source, "created")
		delete(v.Source, "updated")
		err = mapstructure.Decode(v.Source, &role)
		if err != nil {
			log.Error(err)
			return
		}
		core.RoleMap[role.Name] = role
	}

}
