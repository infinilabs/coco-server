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
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/console/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/credential"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"net/http"
	"strconv"
)

type APIHandler struct {
	core.Handler
}

func (h *APIHandler) createCredential(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	cred := credential.Credential{}
	err := h.DecodeJSON(req, &cred)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = cred.Validate()
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = cred.Encode()
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := orm.Context{
		Refresh: "wait_for",
	}
	cred.ID = util.GetUUID()
	err = orm.Create(&ctx, &cred)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteCreatedOKJSON(w, cred.ID)
}

func (h *APIHandler) updateCredential(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")
	obj := credential.Credential{}

	obj.ID = id
	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	newObj := credential.Credential{}
	err = h.DecodeJSON(req, &newObj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}
	err = newObj.Validate()
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	encodeChanged := false
	if obj.Type != newObj.Type {
		encodeChanged = true
	} else {
		switch newObj.Type {
		case credential.BasicAuth:
			var oldPwd string
			if oldParams, ok := obj.Payload[newObj.Type].(map[string]interface{}); ok {
				if pwd, ok := oldParams["password"].(string); ok {
					oldPwd = pwd
				} else {
					http.Error(w, fmt.Sprintf("invalid password of credential [%s]", obj.ID), http.StatusInternalServerError)
					return
				}
			}
			if params, ok := newObj.Payload[newObj.Type].(map[string]interface{}); ok {
				if pwd, ok := params["password"].(string); ok && pwd != oldPwd {
					obj.Payload = newObj.Payload
					encodeChanged = true
				} else {
					if oldParams, ok := obj.Payload[obj.Type].(map[string]interface{}); ok {
						oldParams["username"] = params["username"]
					}
				}
			}
		default:
			h.WriteError(w, fmt.Sprintf("unsupport credential type [%s]", newObj.Type), http.StatusInternalServerError)
			return
		}
	}
	obj.Name = newObj.Name
	obj.Type = newObj.Type
	obj.Tags = newObj.Tags
	if encodeChanged {
		err = obj.Encode()
		if err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			log.Error(err)
			return
		}
	}
	ctx := &orm.Context{
		Refresh: "wait_for",
	}
	obj.Invalid = false
	err = orm.Update(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}
	task.RunWithinGroup("credential_callback", func(ctx context.Context) error {
		credential.TriggerChangeEvent(&obj)
		return nil
	})

	h.WriteUpdatedOKJSON(w, id)
}

func (h *APIHandler) deleteCredential(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj := credential.Credential{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}
	//check dependency
	toDelete, err := canDelete(&obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}
	if !toDelete {
		h.WriteError(w, "This credential is in use and cannot be deleted", http.StatusInternalServerError)
		return
	}
	ctx := &orm.Context{
		Refresh: "wait_for",
	}
	err = orm.Delete(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}

	h.WriteDeletedOKJSON(w, id)
}

func canDelete(cred *credential.Credential) (bool, error) {
	if cred == nil {
		return false, fmt.Errorf("parameter cred can not be nil")
	}
	q := orm.Query{
		Conds: orm.And(orm.Eq("credential_id", cred.ID)),
	}
	err, result := orm.Search(elastic.ElasticsearchConfig{}, &q)
	if err != nil {
		return false, fmt.Errorf("query elasticsearch config error: %w", err)
	}
	return result.Total == 0, nil
}

func (h *APIHandler) searchCredential(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var (
		keyword = h.GetParameterOrDefault(req, "keyword", "")
		strSize = h.GetParameterOrDefault(req, "size", "20")
		strFrom = h.GetParameterOrDefault(req, "from", "0")
		mustQ   []interface{}
	)
	if keyword != "" {
		mustQ = append(mustQ, util.MapStr{
			"query_string": util.MapStr{
				"default_field": "*",
				"query":         keyword,
			},
		})
	}
	size, _ := strconv.Atoi(strSize)
	if size <= 0 {
		size = 20
	}
	from, _ := strconv.Atoi(strFrom)
	if from < 0 {
		from = 0
	}

	queryDSL := util.MapStr{
		"size": size,
		"from": from,
		"sort": []util.MapStr{
			{
				"created": util.MapStr{
					"order": "desc",
				},
			},
		},
	}
	if len(mustQ) > 0 {
		queryDSL["query"] = util.MapStr{
			"bool": util.MapStr{
				"must": mustQ,
			},
		}
	}

	q := orm.Query{}
	q.RawQuery = util.MustToJSONBytes(queryDSL)

	err, res := orm.Search(&credential.Credential{}, &q)
	if err != nil {
		log.Error(err)
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	searchRes := elastic.SearchResponse{}
	util.MustFromJSONBytes(res.Raw, &searchRes)
	if len(searchRes.Hits.Hits) > 0 {
		for _, hit := range searchRes.Hits.Hits {
			delete(hit.Source, "encrypt")
			util.MapStr(hit.Source).Delete("payload.basic_auth.password")
		}
	}

	h.WriteJSON(w, searchRes, http.StatusOK)
}

func (h *APIHandler) getCredential(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")
	obj := credential.Credential{}

	obj.ID = id
	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}
	util.MapStr(obj.Payload).Delete("basic_auth.password")
	h.WriteGetOKJSON(w, id, obj)
}
