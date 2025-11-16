/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import "infini.sh/framework/core/orm"

type Session struct {
	orm.ORMObjectBase
	Status               string `config:"status" json:"status,omitempty" elastic_mapping:"status:{type:keyword}"`
	Title                string `json:"title,omitempty" elastic_mapping:"title:{type:text,copy_to:combined_fulltext,fields:{text: {type: text}, pinyin: {type: text, analyzer: pinyin_analyzer}}}"` // Document title
	Summary              string `config:"summary" json:"summary,omitempty" elastic_mapping:"summary:{type:text}"`
	ManuallyRenamedTitle bool   `config:"manually_renamed_title" json:"manually_renamed_title,omitempty" elastic_mapping:"manually_renamed_title:{type:boolean}"`

	Visible bool `json:"visible" elastic_mapping:"visible:{type:boolean}"` // Whether the connector is enabled or not

	Context *SessionContext `config:"context" json:"context,omitempty" elastic_mapping:"context:{type:object}"`
}

type SessionContext struct {
	Attachments []string `config:"attachments" json:"attachments,omitempty" elastic_mapping:"attachments:{type:keyword}"`
}
