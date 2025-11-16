/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"fmt"
	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

func GetFullPathForCategories(parentCategories []string) string {
	if len(parentCategories) == 0 {
		return "/"
	} else if len(parentCategories) == 1 {
		if parentCategories[0] == "/" {
			return "/"
		}
	}
	path := util.JoinArray(parentCategories, "/")
	if !util.PrefixStr(path, "/") {
		path = "/" + path
	}

	if !util.SuffixStr(path, "/") {
		path = path + "/"
	}
	return path
}

const SystemHierarchyPathKey = "parent_path"

func CreateHierarchyPathFolderDoc(datasource *core.DataSource, id string, name string, parentCategories []string) core.Document {
	document := core.Document{
		Source: core.DataSourceReference{
			ID:   datasource.ID,
			Name: datasource.Name,
			Type: "connector",
		},
		Title: name,
		Type:  "folder",
		Icon:  "font_filetype-folder", //getIcon("folder"),
	}

	document.System = datasource.System
	if document.System == nil {
		document.System = util.MapStr{}
	}
	path := GetFullPathForCategories(parentCategories)
	document.System[SystemHierarchyPathKey] = path
	document.Category = path
	document.Categories = parentCategories

	if id == "" {
		id = util.GetUUID()
	}
	document.ID = GetDocID(datasource.ID, id)
	return document
}

func GetDocID(datasourceID, docID string) string {
	return util.MD5digest(fmt.Sprintf("%v_%v", datasourceID, docID))
}
