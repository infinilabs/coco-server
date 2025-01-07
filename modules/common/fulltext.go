/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import "infini.sh/framework/core/orm"


type CombinedFullText struct {
	orm.ORMObjectBase // Embedding ORM base for persistence-related fields
	CombinedFullText string `json:"-" elastic_mapping:"combined_fulltext:{type:text,index_prefixes:{},index_phrases:true, analyzer:combined_text_analyzer }"`

	Metadata map[string]interface{} `json:"metadata,omitempty" elastic_mapping:"metadata:{type:object}"` // Additional accessible metadata (e.g., file version, permissions)
	Payload  map[string]interface{} `json:"payload,omitempty" elastic_mapping:"payload:{enabled:false}"` // Additional store-only metadata (e.g., file binary data)
}
