/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package salesforce

import (
	"context"
	"fmt"
	"strings"

	"infini.sh/coco/modules/common"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"

	log "github.com/cihub/seelog"
)

const (
	ConnectorSalesforce = "salesforce"
)

type Plugin struct {
	cmn.ConnectorProcessorBase
}

func init() {
	pipeline.RegisterProcessorPlugin(ConnectorSalesforce, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	runner := Plugin{}
	runner.Init(c, &runner)
	return &runner, nil
}

func (p *Plugin) Name() string {
	return ConnectorSalesforce
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

func (p *Plugin) Fetch(ctx *pipeline.Context, connector *common.Connector, datasource *common.DataSource) error {
	cfg := Config{}
	p.MustParseConfig(datasource, &cfg)

	log.Debugf("[%s connector] handling datasource: %v", ConnectorSalesforce, datasource.Name)

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
		return fmt.Errorf("failed to extract OAuth configuration: %v", err)
	}

	// Create client with connector-level OAuth config
	clientConfig := &Config{
		OAuth:                 oauthConfig,
		StandardObjectsToSync: cfg.StandardObjectsToSync,
		SyncCustomObjects:     cfg.SyncCustomObjects,
		CustomObjectsToSync:   cfg.CustomObjectsToSync,
	}
	client := NewSalesforceClient(clientConfig)

	if err := p.processSalesforceData(ctx, client, clientConfig, connector, datasource); err != nil {
		return err
	}

	log.Infof("[%s connector] finished fetching datasource [%s]", ConnectorSalesforce, datasource.Name)
	return nil
}

func (p *Plugin) processSalesforceData(
	ctx *pipeline.Context,
	client *SalesforceClient,
	cfg *Config,
	connector *common.Connector,
	datasource *common.DataSource,
) error {
	// Authenticate with Salesforce
	authCtx := context.Background()
	if err := client.Authenticate(authCtx); err != nil {
		return fmt.Errorf("failed to authenticate with Salesforce: %v", err)
	}

	// Get queryable objects once to avoid repeated API calls
	queryableObjects, err := client.GetQueryableSObjects(authCtx)
	if err != nil {
		return fmt.Errorf("failed to get queryable objects: %v", err)
	}

	log.Infof(
		"[%s connector] found %d queryable objects: %v",
		ConnectorSalesforce,
		len(queryableObjects),
		queryableObjects,
	)

	// Create SObject type directories
	if err := p.createSObjectDirectories(ctx, authCtx, client, cfg, connector, datasource); err != nil {
		return err
	}

	// Process standard objects
	for _, objType := range StandardSObjects {
		if len(cfg.StandardObjectsToSync) == 0 || contains(cfg.StandardObjectsToSync, objType) {
			// Check if the object is queryable before attempting to query
			isQueryable, err := client.IsQueryable(authCtx, objType)
			if err != nil {
				return fmt.Errorf("failed to check if object %s is queryable: %v", objType, err)
			}

			if !isQueryable {
				log.Warnf(
					"[%s connector] object %s is not queryable, skipping",
					ConnectorSalesforce,
					objType,
				)
				continue
			}
			if err := p.processObjectType(ctx, authCtx, client, objType, connector, datasource); err != nil {
				return err
			}
		}
	}

	// Process custom objects if enabled
	if cfg.SyncCustomObjects {
		if err := p.processCustomObjects(ctx, authCtx, client, cfg, connector, datasource); err != nil {
			return err
		}
	}

	return nil
}

func (p *Plugin) processCustomObjects(
	ctx *pipeline.Context,
	authCtx context.Context,
	client *SalesforceClient,
	cfg *Config,
	connector *common.Connector,
	datasource *common.DataSource,
) error {
	// Determine which custom objects to sync
	var customObjectsToSync []string
	if len(cfg.CustomObjectsToSync) == 0 {
		log.Infof("[%s connector] sync custom objects is enabled, but no custom objects", ConnectorSalesforce)
		return nil
	} else if len(cfg.CustomObjectsToSync) == 1 && cfg.StandardObjectsToSync[0] == "*" {
		// Sync all custom objects
		customObjects, err := client.GetCustomObjects(authCtx)
		if err != nil {
			return fmt.Errorf("failed to get custom objects: %v", err)
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
		if err := p.processObjectType(ctx, authCtx, client, customObject, connector, datasource); err != nil {
			return err
		}
	}

	return nil
}

func (p *Plugin) processObjectType(
	ctx *pipeline.Context,
	authCtx context.Context,
	client *SalesforceClient,
	objType string,
	connector *common.Connector,
	datasource *common.DataSource,
) error {
	log.Debugf("[%s connector] processing object type: %s", ConnectorSalesforce, objType)

	// Special handling for Case objects to include Feeds
	if objType == "Case" {
		return p.processCaseWithFeeds(ctx, authCtx, client, connector, datasource)
	}

	// Query the object
	records, err := client.QueryObject(authCtx, objType)
	if err != nil {
		return fmt.Errorf("failed to query object %s: %v", objType, err)
	}

	// Convert and collect each record with proper hierarchy path
	for _, record := range records {
		doc := convertToDocumentWithHierarchy(record, objType, datasource, client.instanceUrl)
		if doc != nil {
			p.Collect(ctx, connector, datasource, *doc)
		}
	}

	log.Debugf(
		"[%s connector] processed %d records for object type: %s",
		ConnectorSalesforce,
		len(records),
		objType,
	)

	return nil
}

func (p *Plugin) processCaseWithFeeds(
	ctx *pipeline.Context,
	authCtx context.Context,
	client *SalesforceClient,
	connector *common.Connector,
	datasource *common.DataSource,
) error {
	log.Debugf("[%s connector] processing Case objects with Feeds", ConnectorSalesforce)

	// Query Case records
	records, err := client.QueryObject(authCtx, "Case")
	if err != nil {
		return fmt.Errorf("failed to query Case objects: %v", err)
	}

	// Check if CaseFeed is queryable
	caseFeedsByCaseId := make(map[string][]map[string]interface{})
	if len(records) > 0 {
		// Check if CaseFeed is queryable
		caseFeedQueryable, err := client.IsQueryable(authCtx, "CaseFeed")
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
				caseFeedsByCaseId = p.getCaseFeedsByCaseId(authCtx, client, allCaseIds)
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

		// Convert and collect the record with proper hierarchy path
		doc := convertToDocumentWithHierarchy(record, "Case", datasource, client.instanceUrl)
		if doc != nil {
			p.Collect(ctx, connector, datasource, *doc)
		}
	}

	log.Debugf(
		"[%s connector] processed %d Case records with Feeds",
		ConnectorSalesforce,
		len(records),
	)

	return nil
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

// createSObjectDirectories creates directory structure for SObject types
func (p *Plugin) createSObjectDirectories(
	ctx *pipeline.Context,
	authCtx context.Context,
	client *SalesforceClient,
	cfg *Config,
	connector *common.Connector,
	datasource *common.DataSource,
) error {
	// Create Standard Objects directory
	standardObjectsDoc := common.CreateHierarchyPathFolderDoc(
		datasource,
		"standard_objects",
		"Standard Objects",
		[]string{},
	)
	standardObjectsDoc.URL = fmt.Sprintf("https://%s.my.salesforce.com", datasource.Name)
	p.Collect(ctx, connector, datasource, standardObjectsDoc)

	// Create Custom Objects directory if custom objects are enabled
	if cfg.SyncCustomObjects {
		customObjectsDoc := common.CreateHierarchyPathFolderDoc(
			datasource,
			"custom_objects",
			"Custom Objects",
			[]string{},
		)
		customObjectsDoc.URL = fmt.Sprintf("https://%s.my.salesforce.com", datasource.Name)
		p.Collect(ctx, connector, datasource, customObjectsDoc)
	}

	// Create directories for each SObject type
	allObjects := make([]string, 0)

	// Add standard objects
	for _, objType := range StandardSObjects {
		if len(cfg.StandardObjectsToSync) == 0 || contains(cfg.StandardObjectsToSync, objType) {
			// Check if the object is queryable
			isQueryable, err := client.IsQueryable(authCtx, objType)
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
			customObjects, err := client.GetCustomObjects(authCtx)
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
		p.Collect(ctx, connector, datasource, sobjectDoc)

		log.Debugf("[%s connector] created directory for SObject type: %s", ConnectorSalesforce, objType)
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
