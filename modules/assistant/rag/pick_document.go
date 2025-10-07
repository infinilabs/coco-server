/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rag

import "infini.sh/framework/core/util"

type PickedDocument struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Explain string `json:"explain"`
}

func PickedDocumentFromString(str string) ([]PickedDocument, error) {
	str = util.TrimLeftStr(str, "<JSON>")
	str = util.TrimRightStr(str, "</JSON>")
	str = util.TrimSpaces(str)
	obj := []PickedDocument{}
	err := util.FromJSONBytes([]byte(str), &obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
