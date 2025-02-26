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

package api

import (
	"infini.sh/coco/plugins/security/core"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
	"net/http"
	log "src/github.com/cihub/seelog"
	"time"
)

func (h APIHandler) CreateRole(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	roleType := ps.MustGetParameter("type")

	//localUser, err := core.FromUserContext(r.Context())
	//if err != nil {
	//	log.Error(err.Error())
	//	h.ErrorInternalServer(w, err.Error())
	//	return
	//}
	err := core.IsAllowRoleType(roleType)
	if err != nil {
		h.ErrorInternalServer(w, err.Error())
		return
	}
	role := &core.Role{
		Type: roleType,
	}
	err = h.DecodeJSON(r, role)
	if err != nil {
		h.Error400(w, err.Error())
		return
	}
	if _, ok := core.RoleMap[role.Name]; ok {
		h.ErrorInternalServer(w, "role name already exists")
		return
	}
	now := time.Now()
	role.Created = &now
	role.Updated = role.Created
	role.Type = roleType
	var id string
	id, err = h.Adapter.Role.Create(role)

	if err != nil {
		_ = log.Error(err.Error())
		h.ErrorInternalServer(w, err.Error())
		return
	}
	core.RoleMap[role.Name] = *role
	_ = h.WriteOKJSON(w, api.CreateResponse(id))
	return

}

func (h APIHandler) SearchRole(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	var (
		keyword = h.GetParameterOrDefault(r, "keyword", "")
		from    = h.GetIntOrDefault(r, "from", 0)
		size    = h.GetIntOrDefault(r, "size", 20)
	)

	res, err := h.Adapter.Role.Search(keyword, from, size)
	if err != nil {
		log.Error(err)
		h.ErrorInternalServer(w, err.Error())
		return
	}
	response := elastic.SearchResponse{}
	util.FromJSONBytes(res.Raw, &response)

	hits := response.Hits.Hits
	list := make([]elastic.IndexDocument, 0)
	total := response.GetTotal()
	var index string
	for _, v := range hits {
		index = v.Index
	}
	for k, v := range core.BuiltinRoles {
		mval := map[string]interface{}{}
		vbytes := util.MustToJSONBytes(v)
		util.MustFromJSONBytes(vbytes, &mval)
		list = append(list, elastic.IndexDocument{
			ID:     k,
			Index:  index,
			Type:   "_doc",
			Source: mval,
		})
		total++
	}
	list = append(list, hits...)
	response.Hits.Hits = list
	response.Hits.Total = total

	h.WriteOKJSON(w, response)
	return

}

func (h APIHandler) GetRole(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")
	role, err := h.Adapter.Role.Get(id)

	if err != nil {
		_ = log.Error(err.Error())
		h.ErrorInternalServer(w, err.Error())
		return
	}
	h.WriteOKJSON(w, api.Response{Hit: role})
	return
}

func (h APIHandler) DeleteRole(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	//localUser, err := biz.FromUserContext(r.Context())
	//if err != nil {
	//	log.Error(err.Error())
	//	h.ErrorInternalServer(w, err.Error())
	//	return
	//}
	oldRole, err := h.Role.Get(id)
	if err != nil {
		h.ErrorInternalServer(w, err.Error())
	}
	err = h.Adapter.Role.Delete(id)

	if err != nil {
		_ = log.Error(err.Error())
		h.ErrorInternalServer(w, err.Error())
		return
	}
	delete(core.RoleMap, oldRole.Name)
	_ = h.WriteOKJSON(w, api.DeleteResponse(id))
	return
}

func (h APIHandler) UpdateRole(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")
	//localUser, err := biz.FromUserContext(r.Context())
	//if err != nil {
	//	log.Error(err.Error())
	//	h.ErrorInternalServer(w, err.Error())
	//	return
	//}
	role := &core.Role{}
	err := h.DecodeJSON(r, role)
	if err != nil {
		h.Error400(w, err.Error())
		return
	}
	role.ID = id

	oldRole, err := h.Role.Get(id)
	if err != nil {
		log.Error(err)
		h.ErrorInternalServer(w, err.Error())
		return
	}
	if role.Name != oldRole.Name {
		h.ErrorInternalServer(w, "Changing role name is not allowed")
		return
	}
	now := time.Now()
	role.Type = oldRole.Type
	role.Updated = &now
	role.Created = oldRole.Created
	err = h.Role.Update(role)
	delete(core.RoleMap, oldRole.Name)
	core.RoleMap[role.Name] = *role

	if err != nil {
		_ = log.Error(err.Error())
		h.ErrorInternalServer(w, err.Error())
		return
	}
	_ = h.WriteOKJSON(w, api.UpdateResponse(id))
	return
}
