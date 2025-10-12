/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package dispatcher

import (
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/kv"
	"net/http"
)

const lastAccessTimeKey = "/datasource/lastAccessTime"

func (processor *Dispatcher) resetAccessTime(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//from context
	datasourceID := ps.MustGetParameter("id")

	err := kv.DeleteKey(lastAccessTimeKey, []byte(datasourceID))
	if err != nil {
		panic(err)
	}

	processor.WriteAckOKJSON(w)
}

func (processor *Dispatcher) saveLastAccessTime(datasourceID string, lastModifiedTime string) error {
	err := kv.AddValue(lastAccessTimeKey, []byte(datasourceID), []byte(lastModifiedTime))
	return err
}

func (processor *Dispatcher) getLastAccessTime(datasourceID string) (string, error) {
	data, err := kv.GetValue(lastAccessTimeKey, []byte(datasourceID))
	if err != nil {
		return "", err
	}

	return string(data), nil
}
