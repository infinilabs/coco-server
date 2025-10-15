/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package salesforce

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
)

const (
	ConnectorSalesforce = "salesforce"
)

func init() {
	module.RegisterUserPlugin(&Plugin{})
}

type Plugin struct {
	connectors.BasePlugin
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

func (p *Plugin) Name() string {
	return ConnectorSalesforce
}

func (p *Plugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return p.BasePlugin.Start(connectors.DefaultSyncInterval)
}

func (p *Plugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cancel != nil {
		log.Debugf("[%s connector] received stop signal, cancelling all operations", ConnectorSalesforce)
		p.cancel()
		p.ctx = nil
		p.cancel = nil
	}
	return nil
}

func (p *Plugin) Setup() {
	// Initialize base plugin (handles config parsing and enabled check)
	p.BasePlugin.Init(fmt.Sprintf("connector.%s", ConnectorSalesforce), "indexing salesforce data", p)
}

// extractOAuthConfig extracts OAuth configuration from connector.Config
func (p *Plugin) extractOAuthConfig(connectorConfig map[string]interface{}) (OAuthConfig, error) {
	oauthConfig := OAuthConfig{}

	// Extract OAuth configuration directly from connector.Config
	if domain, ok := connectorConfig["domain"].(string); ok {
		oauthConfig.Domain = domain
	}
	if clientID, ok := connectorConfig["client_id"].(string); ok {
		oauthConfig.ClientID = clientID
	}
	if clientSecret, ok := connectorConfig["client_secret"].(string); ok {
		oauthConfig.ClientSecret = clientSecret
	}

	// Validate required fields
	if oauthConfig.Domain == "" {
		return oauthConfig, fmt.Errorf("domain is required for connector")
	}
	if oauthConfig.ClientID == "" {
		return oauthConfig, fmt.Errorf("client_id is required for connector")
	}
	if oauthConfig.ClientSecret == "" {
		return oauthConfig, fmt.Errorf("client_secret is required for connector")
	}

	return oauthConfig, nil
}

func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	p.mu.Lock()
	parentCtx := p.ctx
	p.mu.Unlock()

	if parentCtx == nil {
		_ = log.Warnf(
			"[%s connector] plugin is stopped, skipping scan for datasource [%s]",
			ConnectorSalesforce,
			datasource.Name,
		)
		return
	}

	cfg := Config{}
	if err := connectors.ParseConnectorConfigure(connector, datasource, &cfg); err != nil {
		_ = log.Errorf(
			"[%s connector] parsing connector configuration failed for datasource [%s]: %v",
			ConnectorSalesforce,
			datasource.Name,
			err,
		)
		return
	}

	// Set default values if not configured
	if cfg.StandardObjectsToSync == nil {
		cfg.StandardObjectsToSync = StandardSObjects
	}
	if cfg.CustomObjectsToSync == nil {
		cfg.CustomObjectsToSync = []string{}
	}

	// Extract OAuth configuration from connector.Config
	oauthConfig, err := p.extractOAuthConfig(connector.Config)
	if err != nil {
		_ = log.Errorf(
			"[%s connector] failed to extract OAuth configuration: %v",
			ConnectorSalesforce,
			err,
		)
		return
	}

	// Create client with connector-level OAuth config
	clientConfig := &Config{
		OAuth:                 oauthConfig,
		StandardObjectsToSync: cfg.StandardObjectsToSync,
		SyncCustomObjects:     cfg.SyncCustomObjects,
		CustomObjectsToSync:   cfg.CustomObjectsToSync,
	}
	client := NewSalesforceClient(clientConfig)

	scanCtx, scanCancel := context.WithCancel(parentCtx)
	defer scanCancel()

	p.processSalesforceData(scanCtx, client, clientConfig, datasource)

	log.Infof(
		"[%s connector] finished scanning datasource [%s]",
		ConnectorSalesforce,
		datasource.Name,
	)
}

func (p *Plugin) processSalesforceData(
	ctx context.Context,
	client *SalesforceClient,
	cfg *Config,
	datasource *common.DataSource,
) {
	// Authenticate with Salesforce
	if err := client.Authenticate(ctx); err != nil {
		_ = log.Errorf(
			"[%s connector] failed to authenticate with Salesforce: %v",
			ConnectorSalesforce,
			err,
		)
		return
	}

	// Get queryable objects once to avoid repeated API calls
	queryableObjects, err := client.GetQueryableSObjects(ctx)
	if err != nil {
		_ = log.Errorf(
			"[%s connector] failed to get queryable objects: %v",
			ConnectorSalesforce,
			err,
		)
		return
	}

	log.Infof(
		"[%s connector] found %d queryable objects: %v",
		ConnectorSalesforce,
		len(queryableObjects),
		queryableObjects,
	)

	// Create SObject type directories
	p.createSObjectDirectories(ctx, client, cfg, datasource)

	// Process standard objects
	for _, objType := range StandardSObjects {
		if len(cfg.StandardObjectsToSync) == 0 || contains(cfg.StandardObjectsToSync, objType) {
			// Check if the object is queryable before attempting to query
			isQueryable, err := client.IsQueryable(ctx, objType)
			if err != nil {
				_ = log.Errorf(
					"[%s connector] failed to check if object %s is queryable: %v",
					ConnectorSalesforce,
					objType,
					err,
				)
				return
			}

			if !isQueryable {
				log.Warnf(
					"[%s connector] object %s is not queryable, skipping",
					ConnectorSalesforce,
					objType,
				)
				return
			}
			p.processObjectType(ctx, client, objType, datasource)
		}
	}

	// Process custom objects if enabled
	if cfg.SyncCustomObjects {
		p.processCustomObjects(ctx, client, cfg, datasource)
	}
}

func (p *Plugin) processCustomObjects(
	ctx context.Context,
	client *SalesforceClient,
	cfg *Config,
	datasource *common.DataSource,
) {
	// Determine which custom objects to sync
	var customObjectsToSync []string
	if len(cfg.CustomObjectsToSync) == 0 {
		log.Infof("[%s connector] sync custom objects is enabled, but no custom objects", ConnectorSalesforce)
		return
	} else if len(cfg.CustomObjectsToSync) == 1 && cfg.StandardObjectsToSync[0] == "*" {
		// Sync all custom objects
		customObjects, err := client.GetCustomObjects(ctx)
		if err != nil {
			_ = log.Errorf(
				"[%s connector] failed to get custom objects: %v",
				ConnectorSalesforce,
				err,
			)
			return
		}
		customObjectsToSync = customObjects
		log.Infof(
			"[%s connector] fetching all custom objects: %v",
			ConnectorSalesforce,
			customObjectsToSync,
		)
	} else {
		// Sync configured custom objects
		customObjectsToSync = cfg.CustomObjectsToSync
		log.Infof(
			"[%s connector] fetching configured custom objects: %v",
			ConnectorSalesforce,
			customObjectsToSync,
		)
	}

	// Process each custom object
	for _, customObject := range customObjectsToSync {
		p.processObjectType(ctx, client, customObject, datasource)
	}
}

func (p *Plugin) processObjectType(
	ctx context.Context,
	client *SalesforceClient,
	objType string,
	datasource *common.DataSource,
) {
	log.Debugf("[%s connector] processing object type: %s", ConnectorSalesforce, objType)

	// Special handling for Case objects to include Feeds
	if objType == "Case" {
		p.processCaseWithFeeds(ctx, client, datasource)
		return
	}

	// Query the object
	records, err := client.QueryObject(ctx, objType)
	if err != nil {
		_ = log.Errorf(
			"[%s connector] failed to query object %s: %v",
			ConnectorSalesforce,
			objType,
			err,
		)
		return
	}

	// Convert and index each record with proper hierarchy path
	for _, record := range records {
		doc := convertToDocumentWithHierarchy(record, objType, datasource, client.instanceUrl)
		if doc != nil {
			p.indexDocument(doc)
		}
	}

	log.Debugf(
		"[%s connector] processed %d records for object type: %s",
		ConnectorSalesforce,
		len(records),
		objType,
	)
}

func (p *Plugin) processCaseWithFeeds(
	ctx context.Context,
	client *SalesforceClient,
	datasource *common.DataSource,
) {
	log.Debugf("[%s connector] processing Case objects with Feeds", ConnectorSalesforce)

	// Query Case records
	records, err := client.QueryObject(ctx, "Case")
	if err != nil {
		_ = log.Errorf(
			"[%s connector] failed to query Case objects: %v",
			ConnectorSalesforce,
			err,
		)
		return
	}

	// Check if CaseFeed is queryable
	caseFeedsByCaseId := make(map[string][]map[string]interface{})
	if len(records) > 0 {
		// Check if CaseFeed is queryable
		caseFeedQueryable, err := client.IsQueryable(ctx, "CaseFeed")
		if err != nil {
			log.Warnf(
				"[%s connector] failed to check if CaseFeed is queryable: %v",
				ConnectorSalesforce,
				err,
			)
		} else if caseFeedQueryable {
			// Get all case IDs
			var allCaseIds []string
			for _, record := range records {
				if id, ok := record["Id"].(string); ok {
					allCaseIds = append(allCaseIds, id)
				}
			}

			// Query case feeds in batches
			if len(allCaseIds) > 0 {
				caseFeedsByCaseId = p.getCaseFeedsByCaseId(ctx, client, allCaseIds)
			}
		}
	}

	// Process each case record with its feeds
	for _, record := range records {
		// Add feeds to the record
		if caseId, ok := record["Id"].(string); ok {
			if feeds, exists := caseFeedsByCaseId[caseId]; exists {
				record["Feeds"] = feeds
			}
		}

		// Convert and index the record with proper hierarchy path
		doc := convertToDocumentWithHierarchy(record, "Case", datasource, client.instanceUrl)
		if doc != nil {
			p.indexDocument(doc)
		}
	}

	log.Debugf(
		"[%s connector] processed %d Case records with Feeds",
		ConnectorSalesforce,
		len(records),
	)
}

func (p *Plugin) getCaseFeedsByCaseId(
	ctx context.Context,
	client *SalesforceClient,
	caseIds []string,
) map[string][]map[string]interface{} {
	caseFeedsByCaseId := make(map[string][]map[string]interface{})

	// Process case IDs in batches of 800
	batchSize := 800
	for i := 0; i < len(caseIds); i += batchSize {
		end := i + batchSize
		if end > len(caseIds) {
			end = len(caseIds)
		}

		batchCaseIds := caseIds[i:end]

		// Query case feeds for this batch
		query, err := client.caseFeedsQuery(ctx, batchCaseIds)
		if err != nil {
			log.Warnf(
				"[%s connector] failed to query case feeds for batch: %v",
				ConnectorSalesforce,
				err,
			)
			continue
		}
		feeds, err := client.QueryWithSOQL(ctx, query)
		if err != nil {
			log.Warnf(
				"[%s connector] failed to query case feeds for batch: %v",
				ConnectorSalesforce,
				err,
			)
			continue
		}

		// Group feeds by ParentId (Case ID)
		for _, feed := range feeds {
			if parentId, ok := feed["ParentId"].(string); ok {
				caseFeedsByCaseId[parentId] = append(caseFeedsByCaseId[parentId], feed)
			}
		}
	}

	return caseFeedsByCaseId
}

func (p *Plugin) indexDocument(doc *common.Document) {
	if doc == nil {
		return
	}

	// Convert document to JSON bytes
	data := util.MustToJSONBytes(doc)

	// Push document to queue for indexing
	if err := queue.Push(p.Queue, data); err != nil {
		_ = log.Errorf(
			"[%s connector] failed to push document to queue: %v",
			ConnectorSalesforce,
			err,
		)
		return
	}

	log.Debugf(
		"[%s connector] successfully queued document: %s",
		ConnectorSalesforce,
		doc.Title,
	)
}

// createSObjectDirectories creates directory structure for SObject types
func (p *Plugin) createSObjectDirectories(
	ctx context.Context,
	client *SalesforceClient,
	cfg *Config,
	datasource *common.DataSource,
) {
	// Create Standard Objects directory
	standardObjectsDoc := common.CreateHierarchyPathFolderDoc(
		datasource,
		"standard_objects",
		"Standard Objects",
		[]string{},
	)
	standardObjectsDoc.URL = fmt.Sprintf("https://%s.my.salesforce.com", datasource.Name)
	p.indexDocument(&standardObjectsDoc)

	// Create Custom Objects directory if custom objects are enabled
	if cfg.SyncCustomObjects {
		customObjectsDoc := common.CreateHierarchyPathFolderDoc(
			datasource,
			"custom_objects",
			"Custom Objects",
			[]string{},
		)
		customObjectsDoc.URL = fmt.Sprintf("https://%s.my.salesforce.com", datasource.Name)
		p.indexDocument(&customObjectsDoc)
	}

	// Create directories for each SObject type
	allObjects := make([]string, 0)

	// Add standard objects
	for _, objType := range StandardSObjects {
		if len(cfg.StandardObjectsToSync) == 0 || contains(cfg.StandardObjectsToSync, objType) {
			// Check if the object is queryable
			isQueryable, err := client.IsQueryable(ctx, objType)
			if err != nil {
				log.Warnf(
					"[%s connector] failed to check if object %s is queryable: %v",
					ConnectorSalesforce,
					objType,
					err,
				)
				continue
			}
			if isQueryable {
				allObjects = append(allObjects, objType)
			}
		}
	}

	// Add custom objects if enabled
	if cfg.SyncCustomObjects {
		var customObjectsToSync []string
		if len(cfg.CustomObjectsToSync) == 0 {
			log.Infof("[%s connector] sync custom objects is enabled, but no custom objects", ConnectorSalesforce)
		} else if len(cfg.CustomObjectsToSync) == 1 && cfg.CustomObjectsToSync[0] == "*" {
			// Sync all custom objects
			customObjects, err := client.GetCustomObjects(ctx)
			if err != nil {
				log.Errorf(
					"[%s connector] failed to get custom objects: %v",
					ConnectorSalesforce,
					err,
				)
			} else {
				customObjectsToSync = customObjects
			}
		} else {
			// Sync configured custom objects
			customObjectsToSync = cfg.CustomObjectsToSync
		}
		allObjects = append(allObjects, customObjectsToSync...)
	}

	// Create directory for each SObject type
	for _, objType := range allObjects {
		var parentPath []string
		var objTypeDisplay string

		// Determine if it's a standard or custom object
		isStandard := false
		for _, stdObj := range StandardSObjects {
			if strings.EqualFold(stdObj, objType) {
				isStandard = true
				break
			}
		}

		if isStandard {
			parentPath = []string{"Standard Objects"}
			objTypeDisplay = objType
		} else {
			parentPath = []string{"Custom Objects"}
			objTypeDisplay = objType
		}

		// Create SObject type directory
		sobjectDoc := common.CreateHierarchyPathFolderDoc(
			datasource,
			fmt.Sprintf("sobject_%s", strings.ToLower(objType)),
			objTypeDisplay,
			parentPath,
		)
		sobjectDoc.URL = fmt.Sprintf("https://%s.my.salesforce.com", datasource.Name)
		sobjectDoc.Metadata = map[string]interface{}{
			"sobject_type": objType,
			"is_standard":  isStandard,
		}
		p.indexDocument(&sobjectDoc)

		log.Debugf("[%s connector] created directory for SObject type: %s", ConnectorSalesforce, objType)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
