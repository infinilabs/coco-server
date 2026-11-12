/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"strings"
	"time"

	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/task/chrono"
)

type DataSource struct {
	CombinedFullText

	Type        string `json:"type,omitempty" elastic_mapping:"type:{type:keyword,copy_to:combined_fulltext}"` // Type of the datasource, eg: connector
	Name        string `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext,fields:{text: {type: text}, pinyin: {type: text, analyzer: pinyin_analyzer}}}"`
	Description string `json:"description,omitempty" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"`
	Icon        string `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"` // Display name of this datasource

	Category string   `json:"category,omitempty" elastic_mapping:"category:{type:keyword}"`
	Tags     []string `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword}"`

	// TagsVocab is the controlled vocabulary (W4): when set, the
	// extract_tags processor may ONLY pick from this list — unknown model
	// outputs are dropped, manual tags are never touched. Empty = free-form
	// tagging (the historic behavior).
	TagsVocab []string `json:"tags_vocab,omitempty" elastic_mapping:"tags_vocab:{type:keyword}"`

	Connector ConnectorConfig `json:"connector,omitempty" elastic_mapping:"connector:{type:object}"` // Connector configuration

	// Whether synchronization is allowed
	SyncConfig SyncConfig `json:"sync" elastic_mapping:"sync:{type:object}"`
	Enabled    bool       `json:"enabled" elastic_mapping:"enabled:{type:boolean}"`

	// Enrichment pipeline
	EnrichmentPipeline *pipeline.PipelineConfigV2 `json:"enrichment_pipeline" elastic_mapping:"enrichment_pipeline:{type:object}"` //if the pipeline is enabled, pass each batch messages to this pipeline for enrichment

	WebhookConfig WebhookConfig `json:"webhook,omitempty" elastic_mapping:"webhook:{type:object}"`

	//OAuthConfig OAuthConfig `json:"oauth_config,omitempty" elastic_mapping:"oauth_config:{type:object}"`

	// For the documents that come from this data source, process it in this way.
	DocumentProcessingConfig ProcessingConfig `json:"document_processing_config,omitempty" elastic_mapping:"document_processing_config:{type:object}"`
}

type WebhookConfig struct {
	Enabled bool `json:"enabled" elastic_mapping:"enabled:{type:boolean}"`
}

type OAuthConfig struct {
	Enabled bool `json:"enabled,omitempty" elastic_mapping:"enabled:{type:boolean}"`
	Expired bool `json:"expired" elastic_mapping:"expired:{type:boolean}"`
}

type SyncConfig struct {
	Enabled  bool   `json:"enabled" elastic_mapping:"enabled:{type:boolean}"`
	Strategy string `json:"strategy" elastic_mapping:"strategy:{type:keyword}"`
	Interval string `json:"interval" elastic_mapping:"interval:{type:keyword}"`
	Crontab  string `json:"crontab,omitempty" elastic_mapping:"crontab:{type:keyword}"`
	PageSize int    `json:"page_size" config:"page_size"`
}

// Sync strategies: how a datasource is scheduled into runs. The dispatcher
// branches on the effective value; the webhook handler drives realtime
// sources by push.
const (
	// SyncStrategyInterval auto-dispatches on the configured interval (the
	// historical default; empty or unknown values normalize to this).
	SyncStrategyInterval = "interval"
	// SyncStrategyManual never auto-dispatches — runs happen only through
	// POST /datasource/:id/_sync.
	SyncStrategyManual = "manual"
	// SyncStrategyRealtime is push-driven: webhook deliveries carry the
	// content, no interval polling.
	SyncStrategyRealtime = "realtime"
	// SyncStrategyCron auto-dispatches when a crontab slot has passed since
	// the last run — e.g. "0 0 0 * * *" for every midnight. Crontabs use the
	// framework's chrono 6-field format (second minute hour day-of-month
	// month day-of-week). A missed slot (server down at fire time) runs once
	// on the next tick, because the check anchors on the last run's time.
	SyncStrategyCron = "cron"
)

// EffectiveStrategy normalizes the configured strategy: empty or unknown
// values read as interval so existing records keep their behavior.
func (c SyncConfig) EffectiveStrategy() string {
	switch c.Strategy {
	case SyncStrategyManual, SyncStrategyRealtime, SyncStrategyCron:
		return c.Strategy
	default:
		return SyncStrategyInterval
	}
}

// CronDue reports whether a cron-scheduled source should dispatch now: a
// trigger slot has passed since lastAccess (exclusive) and is due at or
// before now. lastAccess empty or unparsable (never ran / corrupt stamp)
// means due immediately — the first sync should not wait for a slot. An
// invalid or empty crontab is never due (the dispatcher logs it; manual
// trigger still works). Parses the stamp locally instead of calling
// util.ParseTimeWithLocalTZ, which panics on malformed input.
func (c SyncConfig) CronDue(lastAccess string, now time.Time) bool {
	expr, err := chrono.ParseCronExpression(strings.TrimSpace(c.Crontab))
	if err != nil {
		return false
	}
	lastAccess = strings.TrimSpace(lastAccess)
	if lastAccess == "" {
		return true
	}
	last, err := time.ParseInLocation("2006-01-02 15:04:05", lastAccess, time.Local)
	if err != nil {
		return true
	}
	return !expr.NextTime(last).After(now)
}

type ConnectorConfig struct {
	ConnectorID string      `json:"id,omitempty" elastic_mapping:"id:{type:keyword}"`          // Connector ID for the datasource
	Config      interface{} `json:"config,omitempty" elastic_mapping:"config:{enabled:false}"` // Configs for this Connector, also pass to connector's processor
}

// If enabled, run the specified pipeline to process the input objects.
//
// Currently, object can be:
//
// * core.Document
// * core.Attachment
type ProcessingConfig struct {
	Enabled  bool   `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`
	Pipeline string `json:"pipeline" elastic_mapping:"pipeline:{type:keyword}"`
}
