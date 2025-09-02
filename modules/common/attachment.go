/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import "infini.sh/framework/core/orm"

type Attachment struct {
	orm.ORMObjectBase // Embedding ORM base for persistence-related fields

	Name        string `json:"name,omitempty" elastic_mapping:"name:{type:keyword}"`
	Description string `json:"description,omitempty" elastic_mapping:"description:{type:text}"`
	Icon        string `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"`
	MimeType    string `json:"mime_type,omitempty" elastic_mapping:"mime_type:{enabled:false}"`
	URL         string `json:"url,omitempty" elastic_mapping:"url:{enabled:false}"`
	Size        int    `json:"size,omitempty" elastic_mapping:"size:{type:long}"`

	Deleted       bool        `json:"deleted,omitempty" elastic_mapping:"deleted:{type:boolean}"`
	LastUpdatedBy *EditorInfo `json:"last_updated_by,omitempty" elastic_mapping:"last_updated_by:{type:object}"`

	//Owner *UserInfo `json:"owner,omitempty" elastic_mapping:"owner:{type:object}"` // Document author or owner

	Metadata map[string]interface{} `json:"metadata,omitempty" elastic_mapping:"metadata:{type:object}"` // Additional accessible metadata (e.g., file version, permissions)
	Payload  map[string]interface{} `json:"payload,omitempty" elastic_mapping:"payload:{enabled:false}"` // Additional store-only metadata (e.g., file binary data)
}
