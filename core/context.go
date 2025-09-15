/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"errors"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
)

const Secret = "coco"

var secretKey string

func GetSecret() (string, error) {

	if secretKey != "" {
		return secretKey, nil
	}

	exists, err := kv.ExistsKey("Coco", []byte(Secret))
	if err != nil {
		return "", err
	}
	if !exists {
		key := util.GetUUID()
		err = kv.AddValue("Coco", []byte(Secret), []byte(key))
		if err != nil {
			return "", err
		}
		secretKey = key
	} else {
		v, err := kv.GetValue("Coco", []byte(Secret))
		if err != nil {
			return "", err
		}
		if len(v) > 0 {
			secretKey = string(v)
		}
	}

	if secretKey == "" {
		return "", errors.New("invalid secret: unable to create or retrieve secret key")
	}

	return secretKey, nil
}

func RewriteQueryWithFilter(queryDsl []byte, filter util.MapStr) ([]byte, error) {

	mapObj := util.MapStr{}
	err := util.FromJSONBytes(queryDsl, &mapObj)
	if err != nil {
		return nil, err
	}
	must := []util.MapStr{
		filter,
	}
	filterQ := util.MapStr{
		"bool": util.MapStr{
			"must": must,
		},
	}
	v, ok := mapObj["query"].(map[string]interface{})
	if ok { //exists query
		newQuery := util.MapStr{
			"bool": util.MapStr{
				"filter": filterQ,
				"must":   []interface{}{v},
			},
		}
		mapObj["query"] = newQuery
	} else {
		mapObj["query"] = util.MapStr{
			"bool": util.MapStr{
				"filter": filterQ,
			},
		}
	}
	queryDsl = util.MustToJSONBytes(mapObj)
	return queryDsl, nil
}
