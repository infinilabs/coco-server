/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package dispatcher

import (
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/kv"
	"net/http"
)

const lastModifiedTimeKey = "/datasource/lastModifiedTime"

func (processor *Dispatcher) reset(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//from context
	datasourceID := ps.MustGetParameter("id")

	err := kv.DeleteKey(lastModifiedTimeKey, []byte(datasourceID))
	if err != nil {
		panic(err)
	}

	processor.WriteAckOKJSON(w)
}

func (processor *Dispatcher) saveLastModifiedTime(datasourceID string, lastModifiedTime string) error {
	err := kv.AddValue(lastModifiedTimeKey, []byte(datasourceID), []byte(lastModifiedTime))
	return err
}

func (processor *Dispatcher) getLastModifiedTime(datasourceID string) (string, error) {
	data, err := kv.GetValue(lastModifiedTimeKey, []byte(datasourceID))
	if err != nil {
		return "", err
	}

	return string(data), nil
}
