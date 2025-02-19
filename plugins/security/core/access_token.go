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

/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

//func GenerateAccessToken(user *User) (map[string]interface{}, error) {
//
//	var data map[string]interface{}
//	roles, privilege := user.GetPermissions()
//
//	token1 := jwt.NewWithClaims(jwt.SigningMethodHS256, UserClaims{
//		ShortUser: &ShortUser{
//			Provider: user.AuthProvider,
//			Login:    user.Username,
//			UserId:   user.ID,
//			Roles:    roles,
//		},
//		RegisteredClaims: &jwt.RegisteredClaims{
//			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
//		},
//	})
//
//	tokenString, err := token1.SignedString([]byte(Secret))
//	if tokenString == "" || err != nil {
//		return nil, errors.Errorf("failed to generate access_token for user: %v", user.Username)
//	}
//
//	token := Token{ExpireIn: time.Now().Unix() + 86400}
//	SetUserToken(user.ID, token)
//
//	data = util.MapStr{
//		"access_token": tokenString,
//		"username":     user.Username,
//		"id":           user.ID,
//		"expire_in":    86400,
//		"roles":        roles,
//		"privilege":    privilege,
//	}
//
//	data["status"] = "ok"
//
//	return data, err
//
//}
