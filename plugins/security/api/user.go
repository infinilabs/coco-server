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
	"errors"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/modules/elastic"
	"net/http"
)

//func (h APIHandler) CreateUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
//	var user core.User
//	err := h.DecodeJSON(r, &user)
//	if err != nil {
//		h.Error400(w, err.Error())
//		return
//	}
//	if user.Username == "" {
//		h.Error400(w, "username is required")
//		return
//	}
//	//localUser, err := biz.FromUserContext(r.Context())
//	//if err != nil {
//	//	log.Error(err.Error())
//	//	h.ErrorInternalServer(w, err.Error())
//	//	return
//	//}
//	if h.userNameExists(w, user.Username) {
//		return
//	}
//	randStr := util.GenerateRandomString(8)
//	hash, err := bcrypt.GenerateFromPassword([]byte(randStr), bcrypt.DefaultCost)
//	if err != nil {
//		return
//	}
//	user.Password = string(hash)
//
//	now := time.Now()
//	user.Created = &now
//	user.Updated = &now
//
//	id, err := h.User.Create(&user)
//	user.ID = id
//	if err != nil {
//		_ = log.Error(err.Error())
//		h.ErrorInternalServer(w, err.Error())
//		return
//	}
//	_ = h.WriteOKJSON(w, util.MapStr{
//		"_id":      id,
//		"password": randStr,
//		"result":   "created",
//	})
//	return
//
//}

func (h APIHandler) userNameExists(w http.ResponseWriter, name string) bool {
	u, err := h.User.GetBy("name", name)
	if err != nil {
		_ = log.Error(err.Error())
		h.ErrorInternalServer(w, err.Error())
		return true
	}
	if u != nil {
		h.ErrorInternalServer(w, "user name already exists")
		return true
	}
	return false
}

func (h APIHandler) GetUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")
	user, err := h.User.Get(id)
	if errors.Is(err, elastic.ErrNotFound) {
		h.WriteJSON(w, api.NotFoundResponse(id), http.StatusNotFound)
		return
	}

	if err != nil {
		_ = log.Error(err.Error())
		h.ErrorInternalServer(w, err.Error())
		return
	}
	h.WriteOKJSON(w, api.FoundResponse(id, user))
	return
}

//func (h APIHandler) UpdateUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
//	id := ps.MustGetParameter("id")
//	var user core.User
//	err := h.DecodeJSON(r, &user)
//	if err != nil {
//		_ = log.Error(err.Error())
//		h.Error400(w, err.Error())
//		return
//	}
//	//localUser, err := biz.FromUserContext(r.Context())
//	//if err != nil {
//	//	log.Error(err.Error())
//	//	h.ErrorInternalServer(w, err.Error())
//	//	return
//	//}
//	oldUser, err := h.User.Get(id)
//	if err != nil {
//		_ = log.Error(err.Error())
//		h.ErrorInternalServer(w, err.Error())
//		return
//	}
//	if user.Username != oldUser.Username && h.userNameExists(w, user.Username) {
//		return
//	}
//
//	now := time.Now()
//	user.Updated = &now
//	user.Created = oldUser.Created
//	user.ID = id
//	err = h.User.Update(&user)
//
//	if err != nil {
//		_ = log.Error(err.Error())
//		h.ErrorInternalServer(w, err.Error())
//		return
//	}
//	//let user relogin after roles changed
//	sort.Slice(user.Roles, func(i, j int) bool {
//		return user.Roles[i].ID < user.Roles[j].ID
//	})
//	sort.Slice(oldUser.Roles, func(i, j int) bool {
//		return oldUser.Roles[i].ID < oldUser.Roles[j].ID
//	})
//	changeLog, _ := util.DiffTwoObject(user.Roles, oldUser.Roles)
//	if len(changeLog) > 0 {
//		core.DeleteUserToken(id)
//	}
//	_ = h.WriteOKJSON(w, api.UpdateResponse(id))
//	return
//}

//func (h APIHandler) DeleteUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
//	id := ps.MustGetParameter("id")
//	user, err := core.FromUserContext(r.Context())
//	if err != nil {
//		log.Error("failed to get user from context, err: %v", err)
//		h.WriteError(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//	if user != nil && user.UserId == id {
//		h.WriteError(w, "can not delete yourself", http.StatusInternalServerError)
//		return
//	}
//	err = h.User.Delete(id)
//	if errors.Is(err, elastic.ErrNotFound) {
//		h.WriteJSON(w, api.NotFoundResponse(id), http.StatusNotFound)
//		return
//	}
//	if err != nil {
//		_ = log.Error(err.Error())
//		h.ErrorInternalServer(w, err.Error())
//		return
//	}
//	core.DeleteUserToken(id)
//	_ = h.WriteOKJSON(w, api.DeleteResponse(id))
//	return
//}

//func (h APIHandler) SearchUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
//	var (
//		keyword = h.GetParameterOrDefault(r, "keyword", "")
//		from    = h.GetIntOrDefault(r, "from", 0)
//		size    = h.GetIntOrDefault(r, "size", 20)
//	)
//
//	res, err := h.User.Search(keyword, from, size)
//	if err != nil {
//		log.Error(err.Error())
//		h.ErrorInternalServer(w, err.Error())
//		return
//	}
//	//remove password field
//	hitsBuf := bytes.Buffer{}
//	hitsBuf.Write([]byte("["))
//	jsonparser.ArrayEach(res.Raw, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
//		value = jsonparser.Delete(value, "_source", "password")
//		hitsBuf.Write(value)
//		hitsBuf.Write([]byte(","))
//	}, "hits", "hits")
//	buf := hitsBuf.Bytes()
//	if buf[len(buf)-1] == ',' {
//		buf[len(buf)-1] = ']'
//	} else {
//		hitsBuf.Write([]byte("]"))
//	}
//	res.Raw, err = jsonparser.Set(res.Raw, hitsBuf.Bytes(), "hits", "hits")
//	if err != nil {
//		log.Error(err.Error())
//		h.ErrorInternalServer(w, err.Error())
//		return
//	}
//
//	h.Write(w, res.Raw)
//	return
//
//}

//func (h APIHandler) UpdateUserPassword(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
//	id := ps.MustGetParameter("id")
//	var req = struct {
//		Password string `json:"password"`
//	}{}
//	err := h.DecodeJSON(r, &req)
//	if err != nil {
//		_ = log.Error(err.Error())
//		h.Error400(w, err.Error())
//		return
//	}
//	//localUser, err := biz.FromUserContext(r.Context())
//	//if err != nil {
//	//	log.Error(err.Error())
//	//	h.ErrorInternalServer(w, err.Error())
//	//	return
//	//}
//	user, err := h.User.Get(id)
//	if err != nil {
//		_ = log.Error(err.Error())
//		h.ErrorInternalServer(w, err.Error())
//		return
//	}
//	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
//	if err != nil {
//		return
//	}
//	user.Password = string(hash)
//	//t:=time.Now()
//	//user.Updated =&t
//	err = h.User.Update(&user)
//	if err != nil {
//		_ = log.Error(err.Error())
//		h.ErrorInternalServer(w, err.Error())
//		return
//	}
//	//disable old token to let user login
//	core.DeleteUserToken(id)
//
//	_ = h.WriteOKJSON(w, api.UpdateResponse(id))
//	return
//
//}
