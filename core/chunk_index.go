/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package core

import "infini.sh/framework/core/orm"

// KnowledgeChunk (W3 方案 c): a retrievable chunk as an independent
// index row — the "small" leg of small-to-big retrieval. Chunks are
// indexed and searched directly; the parent (mom) row carries the
// section context and is linked via MomID but excluded from retrieval
// (Available=false). Hit a child, return the mom's context — embedding
// granularity decoupled from context granularity.
//
// Written by the document pipeline alongside the nested chunks (which
// stay on the Document for the processing timeline); this index is the
// retrieval face.
type KnowledgeChunk struct {
	orm.ORMObjectBase

	DocID  string              `json:"doc_id" elastic_mapping:"doc_id:{type:keyword}"`
	Source DataSourceReference `json:"source,omitempty" elastic_mapping:"source:{type:object}"`

	Seq int `json:"seq" elastic_mapping:"seq:{type:integer}"`

	// ChunkType: text | table | faq — mirrors the structured splitter's
	// output; "mom" rows carry the section context.
	ChunkType string `json:"chunk_type" elastic_mapping:"chunk_type:{type:keyword}"`

	// Breadcrumb is the accumulated heading path ("年报 > 财务 > 营收").
	Breadcrumb string `json:"breadcrumb,omitempty" elastic_mapping:"breadcrumb:{type:text}"`

	// Text is the chunk body — the BM25 field.
	Text string `json:"text,omitempty" elastic_mapping:"text:{type:text}"`

	// Quote is the ≤300-rune excerpt for citation display.
	Quote string `json:"quote,omitempty" elastic_mapping:"quote:{type:keyword,index:false}"`

	// Locators: page/slide/sheet-row/section — the precise provenance.
	Locators map[string]interface{} `json:"locators,omitempty" elastic_mapping:"locators:{type:object}"`

	// MomID links to the parent/section row; Available=false on mom rows
	// keeps them out of retrieval while preserving the context chain.
	MomID     string `json:"mom_id,omitempty" elastic_mapping:"mom_id:{type:keyword}"`
	Available bool   `json:"available" elastic_mapping:"available:{type:boolean}"`

	// Embedding mirrors the Document's nested chunk vectors — the engine's
	// knn_dense_float_vector with the same 1024-dim LSH config.
	Embedding Embedding `json:"embedding,omitempty" elastic_mapping:"embedding:{type:object}"`

	// ModelID is the embedding model stamp (W1).
	ModelID string `json:"model_id,omitempty" elastic_mapping:"model_id:{type:keyword}"`

	// ContentHash folds same-content chunks for dedup.
	ContentHash string `json:"content_hash,omitempty" elastic_mapping:"content_hash:{type:keyword}"`
}

const (
	ChunkTypeText  = "text"
	ChunkTypeTable = "table"
	ChunkTypeFAQ   = "faq"
	ChunkTypeMom   = "mom"
)
