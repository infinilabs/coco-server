/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/cihub/seelog"
)

const (
	ApiVersion              = "v59.0"
	TokenEndpoint           = "/services/oauth2/token"
	QueryEndpoint           = "/services/data/%s/query"
	DescribeEndpoint        = "/services/data/%s/sobjects"
	DescribeSObjectEndpoint = "/services/data/%s/sobjects/%s/describe"
)

// SalesforceClient handles communication with the Salesforce API
type SalesforceClient struct {
	config      *Config
	httpClient  *http.Client
	accessToken string
	instanceUrl string
	tokenExpiry time.Time

	// Authentication state to prevent recursion
	authenticating bool

	// Field caching
	queryableSObjects      []string
	queryableSObjectFields map[string][]string

	// Content document links join cache
	contentDocumentLinksJoinQuery string
}

// NewSalesforceClient creates a new Salesforce client
func NewSalesforceClient(config *Config) *SalesforceClient {
	return &SalesforceClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		queryableSObjectFields: make(map[string][]string),
	}
}

// Authenticate authenticates with Salesforce and retrieves an access token
func (c *SalesforceClient) Authenticate(ctx context.Context) error {
	baseURL := fmt.Sprintf("https://%s.my.salesforce.com", c.config.OAuth.Domain)
	tokenURL := baseURL + TokenEndpoint

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.config.OAuth.ClientID)
	data.Set("client_secret", c.config.OAuth.ClientSecret)
	header := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	// Use makeRequest directly to avoid infinite recursion
	resp, err := c.makeRequest(ctx, "POST", tokenURL, strings.NewReader(data.Encode()), header)
	if err != nil {
		return fmt.Errorf("failed to execute token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.instanceUrl = tokenResp.InstanceUrl
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	log.Debugf("[salesforce client] successfully authenticated, token expires at %v", c.tokenExpiry)
	return nil
}

// ensureAuthenticated ensures the client has a valid access token
func (c *SalesforceClient) ensureAuthenticated(ctx context.Context) error {
	if c.accessToken == "" || time.Now().After(c.tokenExpiry.Add(-5*time.Minute)) {
		// Prevent recursive authentication calls
		if c.authenticating {
			return fmt.Errorf("authentication already in progress")
		}
		c.authenticating = true
		defer func() { c.authenticating = false }()
		return c.Authenticate(ctx)
	}
	return nil
}

// makeRequest makes an authenticated HTTP request to Salesforce API
func (c *SalesforceClient) makeRequest(
	ctx context.Context,
	method, url string,
	body io.Reader,
	headers map[string]string,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// makeAuthenticatedRequest makes an authenticated HTTP request with automatic authentication
func (c *SalesforceClient) makeAuthenticatedRequest(
	ctx context.Context,
	method, url string,
	body io.Reader,
	headers map[string]string,
) (*http.Response, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}

	return c.makeRequest(ctx, method, url, body, headers)
}

// executeQuery executes a SOQL query and returns the results with pagination support
func (c *SalesforceClient) executeQuery(
	ctx context.Context,
	query string,
	useAuthenticatedRequest bool,
) ([]map[string]interface{}, error) {
	queryURL := fmt.Sprintf("%s%s", c.instanceUrl, fmt.Sprintf(QueryEndpoint, ApiVersion))

	// Add query parameter to URL
	u, err := url.Parse(queryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query URL: %w", err)
	}

	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()

	// Execute the query using appropriate request method
	var resp *http.Response
	if useAuthenticatedRequest {
		resp, err = c.makeAuthenticatedRequest(ctx, "GET", u.String(), nil, nil)
	} else {
		if err := c.ensureAuthenticated(ctx); err != nil {
			return nil, err
		}
		resp, err = c.makeRequest(ctx, "GET", u.String(), nil, nil)
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var queryResp QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}

	allRecords := queryResp.Records

	// Handle pagination if needed
	for queryResp.NextRecordsUrl != "" {
		nextURL := fmt.Sprintf("%s%s", c.instanceUrl, queryResp.NextRecordsUrl)
		nextResp, err := c.makeRequest(ctx, "GET", nextURL, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch next page: %w", err)
		}
		defer nextResp.Body.Close()

		if nextResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(nextResp.Body)
			return nil, fmt.Errorf("pagination failed with status %d: %s", nextResp.StatusCode, string(body))
		}

		var nextQueryResp QueryResponse
		if err := json.NewDecoder(nextResp.Body).Decode(&nextQueryResp); err != nil {
			return nil, fmt.Errorf("failed to decode paginated response: %w", err)
		}

		allRecords = append(allRecords, nextQueryResp.Records...)
		queryResp.NextRecordsUrl = nextQueryResp.NextRecordsUrl
	}

	return allRecords, nil
}

// QueryObject queries a specific Salesforce object
func (c *SalesforceClient) QueryObject(ctx context.Context, objectType string) ([]map[string]interface{}, error) {
	// Build SOQL query based on object type using field caching
	query, err := c.buildSOQLQuery(ctx, objectType)
	if err != nil {
		return nil, err
	}

	// Use authenticated request for QueryObject
	return c.executeQuery(ctx, query, true)
}

// QueryWithSOQL executes a custom SOQL query and returns the results
func (c *SalesforceClient) QueryWithSOQL(ctx context.Context, query string) ([]map[string]interface{}, error) {
	// Use regular request for QueryWithSOQL (authentication handled internally)
	return c.executeQuery(ctx, query, false)
}

// isCustomObject checks if the object type is a custom object (ends with __c)
func (c *SalesforceClient) isCustomObject(objectType string) bool {
	return strings.HasSuffix(objectType, "__c")
}

// buildSOQLQuery builds a SOQL query for the given object type using field caching
func (c *SalesforceClient) buildSOQLQuery(ctx context.Context, objectType string) (string, error) {
	// Check if it's a custom object (ends with __c)
	if c.isCustomObject(objectType) {
		return c.customObjectQuery(ctx, objectType)
	}

	// Handle standard objects
	switch objectType {
	case "Account":
		return c.accountsQuery(ctx, objectType)
	case "Opportunity":
		return c.opportunitiesQuery(ctx, objectType)
	case "Contact":
		return c.contactsQuery(ctx, objectType)
	case "Lead":
		return c.leadsQuery(ctx, objectType)
	case "Campaign":
		return c.campaignsQuery(ctx, objectType)
	case "Case":
		return c.casesQuery(ctx, objectType)
	default:
		// For other non-custom objects (User, EmailMessage, etc.), treat them as custom objects
		return c.customObjectQuery(ctx, objectType)
	}
}

func (c *SalesforceClient) accountsQuery(ctx context.Context, objectType string) (string, error) {
	desiredFields := []string{"Name", "Description", "BillingAddress", "Type", "Website", "Rating", "Department"}

	// Get queryable fields for this object (only direct fields)
	queryableFields, err := c.SelectQueryableFields(ctx, objectType, desiredFields)
	if err != nil {
		return "", fmt.Errorf("failed to get queryable fields for %s: %w", objectType, err)
	}

	// Build query using SalesforceSoqlBuilder
	builder := NewSalesforceSoqlBuilder(objectType).
		WithId().
		WithDefaultMetafields().
		WithFields(queryableFields).
		WithFields([]string{"Owner.Id", "Owner.Name", "Owner.Email"}).
		WithFields([]string{"Parent.Id", "Parent.Name"})

	// Add opportunities join
	opportunitiesJoin := ""
	opporQueryable, err := c.IsQueryable(ctx, "Opportunities")
	if err != nil {
		log.Warnf("[%s connector] failed to get opportunities join: %v", ConnectorSalesforce, err)
	}
	if opporQueryable {
		queryableJoinFields, err := c.SelectQueryableFields(ctx, "Opportunities", []string{"Name", "StageName"})
		if err != nil {
			log.Warnf("[%s connector] failed to get opportunities join: %v", ConnectorSalesforce, err)
		}
		opportunitiesJoin = NewSalesforceSoqlBuilder("Opportunities").
			WithId().
			WithFields(queryableJoinFields).
			WithOrderBy("CreatedDate DESC").
			WithLimit(1).
			Build()
	}
	if opportunitiesJoin != "" {
		builder.WithJoin(opportunitiesJoin)
	}

	// Add content document links join
	contentLinksJoin, err := c.contentDocumentLinksJoin(ctx)
	if err != nil {
		log.Warnf("[%s connector] failed to get content document links join: %v", ConnectorSalesforce, err)
	} else if contentLinksJoin != "" {
		builder.WithJoin(contentLinksJoin)
	}
	return builder.Build(), nil
}

func (c *SalesforceClient) opportunitiesQuery(ctx context.Context, objectType string) (string, error) {
	desiredFields := []string{"Name", "Description", "StageName"}

	// Get queryable fields for this object (only direct fields)
	queryableFields, err := c.SelectQueryableFields(ctx, objectType, desiredFields)
	if err != nil {
		return "", fmt.Errorf("failed to get queryable fields for %s: %w", objectType, err)
	}

	// Build query using SalesforceSoqlBuilder
	builder := NewSalesforceSoqlBuilder(objectType).
		WithId().
		WithDefaultMetafields().
		WithFields(queryableFields).
		WithFields([]string{"Owner.Id", "Owner.Name", "Owner.Email"})

	// Add content document links join
	contentLinksJoin, err := c.contentDocumentLinksJoin(ctx)
	if err != nil {
		log.Warnf("[%s connector] failed to get content document links join: %v", ConnectorSalesforce, err)
	} else if contentLinksJoin != "" {
		builder.WithJoin(contentLinksJoin)
	}
	return builder.Build(), nil
}

func (c *SalesforceClient) contactsQuery(ctx context.Context, objectType string) (string, error) {
	desiredFields := []string{
		"Name", "Description", "Email", "Phone", "Title", "PhotoUrl", "LeadSource",
		"AccountId", "OwnerId",
	}

	// Get queryable fields for this object (only direct fields)
	queryableFields, err := c.SelectQueryableFields(ctx, objectType, desiredFields)
	if err != nil {
		return "", fmt.Errorf("failed to get queryable fields for %s: %w", objectType, err)
	}

	// Build query using SalesforceSoqlBuilder
	builder := NewSalesforceSoqlBuilder(objectType).
		WithId().
		WithDefaultMetafields().
		WithFields(queryableFields)

	// Add content document links join
	contentLinksJoin, err := c.contentDocumentLinksJoin(ctx)
	if err != nil {
		log.Warnf("[%s connector] failed to get content document links join: %v", ConnectorSalesforce, err)
	} else if contentLinksJoin != "" {
		builder.WithJoin(contentLinksJoin)
	}
	return builder.Build(), nil
}

func (c *SalesforceClient) leadsQuery(ctx context.Context, objectType string) (string, error) {
	desiredFields := []string{
		"Company", "ConvertedAccountId", "ConvertedContactId", "ConvertedDate", "ConvertedOpportunityId",
		"Description", "Email", "LeadSource", "Name", "OwnerId", "Phone", "PhotoUrl", "Rating", "Status", "Title",
	}

	// Get queryable fields for this object (only direct fields)
	queryableFields, err := c.SelectQueryableFields(ctx, objectType, desiredFields)
	if err != nil {
		return "", fmt.Errorf("failed to get queryable fields for %s: %w", objectType, err)
	}

	// Build query using SalesforceSoqlBuilder
	builder := NewSalesforceSoqlBuilder(objectType).
		WithId().
		WithDefaultMetafields().
		WithFields(queryableFields)

	// Add content document links join
	contentLinksJoin, err := c.contentDocumentLinksJoin(ctx)
	if err != nil {
		log.Warnf("[%s connector] failed to get content document links join: %v", ConnectorSalesforce, err)
	} else if contentLinksJoin != "" {
		builder.WithJoin(contentLinksJoin)
	}
	return builder.Build(), nil
}

func (c *SalesforceClient) campaignsQuery(ctx context.Context, objectType string) (string, error) {
	desiredFields := []string{
		"Name", "IsActive", "Type", "Description", "Status", "StartDate", "EndDate",
	}

	// Get queryable fields for this object (only direct fields)
	queryableFields, err := c.SelectQueryableFields(ctx, objectType, desiredFields)
	if err != nil {
		return "", fmt.Errorf("failed to get queryable fields for %s: %w", objectType, err)
	}

	// Build query using SalesforceSoqlBuilder
	builder := NewSalesforceSoqlBuilder(objectType).
		WithId().
		WithDefaultMetafields().
		WithFields(queryableFields).
		WithFields([]string{"Owner.Id", "Owner.Name", "Owner.Email"}).
		WithFields([]string{"Parent.Id", "Parent.Name"})

	// Add content document links join
	contentLinksJoin, err := c.contentDocumentLinksJoin(ctx)
	if err != nil {
		log.Warnf("[%s connector] failed to get content document links join: %v", ConnectorSalesforce, err)
	} else if contentLinksJoin != "" {
		builder.WithJoin(contentLinksJoin)
	}
	return builder.Build(), nil
}

func (c *SalesforceClient) casesQuery(ctx context.Context, objectType string) (string, error) {
	desiredFields := []string{
		"Subject", "Description", "CaseNumber", "Status", "AccountId", "ParentId", "IsClosed", "IsDeleted",
	}

	// Get queryable fields for this object (only direct fields)
	queryableFields, err := c.SelectQueryableFields(ctx, objectType, desiredFields)
	if err != nil {
		return "", fmt.Errorf("failed to get queryable fields for %s: %w", objectType, err)
	}

	// Build query using SalesforceSoqlBuilder
	builder := NewSalesforceSoqlBuilder(objectType).
		WithId().
		WithDefaultMetafields().
		WithFields(queryableFields).
		WithFields([]string{"Owner.Id", "Owner.Name", "Owner.Email"}).
		WithFields([]string{"CreatedBy.Id", "CreatedBy.Name", "CreatedBy.Email"})

	// Add email message join
	emailMessageJoin, err := c.emailMessageJoinQuery(ctx)
	if err != nil {
		log.Warnf("[%s connector] failed to get email message join: %v", ConnectorSalesforce, err)
	} else if emailMessageJoin != "" {
		builder.WithJoin(emailMessageJoin)
	}

	// Add case comments join
	caseCommentsJoin, err := c.caseCommentsJoinQuery(ctx)
	if err != nil {
		log.Warnf("[%s connector] failed to get case comments join: %v", ConnectorSalesforce, err)
	} else if caseCommentsJoin != "" {
		builder.WithJoin(caseCommentsJoin)
	}

	// Add content document links join
	contentLinksJoin, err := c.contentDocumentLinksJoin(ctx)
	if err != nil {
		log.Warnf("[%s connector] failed to get content document links join: %v", ConnectorSalesforce, err)
	} else if contentLinksJoin != "" {
		builder.WithJoin(contentLinksJoin)
	}
	return builder.Build(), nil
}

func (c *SalesforceClient) customObjectQuery(ctx context.Context, objectType string) (string, error) {
	// For custom objects, we don't have predefined desired fields
	// Get all queryable fields for this object
	queryableFields, err := c.SelectQueryableFields(ctx, objectType, []string{})
	if err != nil {
		return "", fmt.Errorf("failed to get queryable fields for %s: %w", objectType, err)
	}

	// Build query using SalesforceSoqlBuilder
	builder := NewSalesforceSoqlBuilder(objectType).
		WithFields(queryableFields)

	// Add content document links join
	contentLinksJoin, err := c.contentDocumentLinksJoin(ctx)
	if err != nil {
		log.Warnf("[%s connector] failed to get content document links join: %v", ConnectorSalesforce, err)
	} else if contentLinksJoin != "" {
		builder.WithJoin(contentLinksJoin)
	}
	return builder.Build(), nil
}

// GetCustomObjects returns all available custom objects from Salesforce
func (c *SalesforceClient) GetCustomObjects(ctx context.Context) ([]string, error) {
	describeURL := fmt.Sprintf("%s%s", c.instanceUrl, fmt.Sprintf(DescribeEndpoint, ApiVersion))

	resp, err := c.makeAuthenticatedRequest(ctx, "GET", describeURL, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("describe request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var describeResp struct {
		SObjects []struct {
			Name   string `json:"name"`
			Custom bool   `json:"custom"`
		} `json:"sobjects"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&describeResp); err != nil {
		return nil, fmt.Errorf("failed to decode describe response: %w", err)
	}

	var customObjects []string
	for _, sobject := range describeResp.SObjects {
		if sobject.Custom && strings.HasSuffix(sobject.Name, "__c") {
			customObjects = append(customObjects, sobject.Name)
		}
	}

	return customObjects, nil
}

// GetQueryableSObjects returns a cached list of queryable SObjects
func (c *SalesforceClient) GetQueryableSObjects(ctx context.Context) ([]string, error) {
	if c.queryableSObjects != nil {
		return c.queryableSObjects, nil
	}

	describeURL := fmt.Sprintf("%s%s", c.instanceUrl, fmt.Sprintf(DescribeEndpoint, ApiVersion))

	resp, err := c.makeAuthenticatedRequest(ctx, "GET", describeURL, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("describe request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var describeResp struct {
		SObjects []struct {
			Name      string `json:"name"`
			Queryable bool   `json:"queryable"`
		} `json:"sobjects"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&describeResp); err != nil {
		return nil, fmt.Errorf("failed to decode describe response: %w", err)
	}

	c.queryableSObjects = []string{}
	relevantSet := make(map[string]bool)
	for _, obj := range RelevantSObjects {
		relevantSet[obj] = true
	}

	for _, sobject := range describeResp.SObjects {
		if sobject.Queryable && relevantSet[sobject.Name] {
			c.queryableSObjects = append(c.queryableSObjects, strings.ToLower(sobject.Name))
		}
	}

	return c.queryableSObjects, nil
}

// GetQueryableSObjectFields returns cached queryable fields for the given SObjects
func (c *SalesforceClient) GetQueryableSObjectFields(
	ctx context.Context,
	relevantObjects []string,
	relevantSObjectFields []string,
) (map[string][]string, error) {
	// Find objects that haven't been cached yet
	var objectsToQuery []string
	for _, obj := range relevantObjects {
		if _, exists := c.queryableSObjectFields[obj]; !exists {
			objectsToQuery = append(objectsToQuery, obj)
		}
	}
	if len(objectsToQuery) == 0 {
		return c.queryableSObjectFields, nil
	}

	// Query fields for each object
	for _, sobject := range objectsToQuery {
		endpoint := fmt.Sprintf(DescribeSObjectEndpoint, ApiVersion, sobject)
		describeURL := fmt.Sprintf("%s%s", c.instanceUrl, endpoint)
		resp, err := c.makeRequest(ctx, "GET", describeURL, nil, nil)
		if err != nil {
			log.Warnf("[salesforce client] failed to execute describe request for %s: %v", sobject, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Warnf("[salesforce client] describe request failed for %s with status %d", sobject, resp.StatusCode)
			continue
		}

		var describeResp struct {
			Fields []struct {
				Name string `json:"name"`
			} `json:"fields"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&describeResp); err != nil {
			resp.Body.Close()
			log.Warnf("[salesforce client] failed to decode describe response for %s: %v", sobject, err)
			continue
		}
		resp.Body.Close()

		// Filter fields based on relevant fields
		var queryableFields []string
		if relevantSObjectFields == nil {
			// Include all fields
			for _, field := range describeResp.Fields {
				queryableFields = append(queryableFields, strings.ToLower(field.Name))
			}
		} else {
			// Filter by relevant fields
			relevantSet := make(map[string]bool)
			for _, field := range relevantSObjectFields {
				relevantSet[field] = true
			}

			for _, field := range describeResp.Fields {
				if relevantSet[field.Name] {
					queryableFields = append(queryableFields, strings.ToLower(field.Name))
				}
			}
		}
		c.queryableSObjectFields[strings.ToLower(sobject)] = queryableFields
	}

	return c.queryableSObjectFields, nil
}

// IsQueryable checks if an SObject is queryable
func (c *SalesforceClient) IsQueryable(ctx context.Context, sobject string) (bool, error) {
	queryableObjects, err := c.GetQueryableSObjects(ctx)
	if err != nil {
		return false, err
	}

	sobjectLower := strings.ToLower(sobject)
	for _, obj := range queryableObjects {
		if obj == sobjectLower {
			return true, nil
		}
	}
	return false, nil
}

// SelectQueryableFields selects queryable fields for an SObject
func (c *SalesforceClient) SelectQueryableFields(
	ctx context.Context,
	sobject string,
	fields []string,
) ([]string, error) {
	var sobjectFields map[string][]string
	var err error

	// Check if it's a relevant SObject
	isRelevant := false
	for _, obj := range RelevantSObjects {
		if strings.ToLower(obj) == strings.ToLower(sobject) {
			isRelevant = true
			break
		}
	}

	if isRelevant {
		sobjectFields, err = c.GetQueryableSObjectFields(ctx, RelevantSObjects, RelevantSObjectFields)
	} else {
		sobjectFields, err = c.GetQueryableSObjectFields(ctx, []string{sobject}, nil)
	}

	if err != nil {
		return nil, err
	}

	queryableFields := sobjectFields[strings.ToLower(sobject)]
	if len(fields) == 0 {
		return queryableFields, nil
	}

	// Filter fields to only include queryable ones
	var filteredFields []string
	for _, field := range fields {
		fieldLower := strings.ToLower(field)
		for _, queryableField := range queryableFields {
			if queryableField == fieldLower {
				filteredFields = append(filteredFields, field)
				break
			}
		}
	}

	return filteredFields, nil
}

// contentDocumentLinksJoin returns the cached content document links join query
func (c *SalesforceClient) contentDocumentLinksJoin(ctx context.Context) (string, error) {
	if c.contentDocumentLinksJoinQuery != "" {
		return c.contentDocumentLinksJoinQuery, nil
	}

	// Check if content-related objects are queryable
	linksQueryable, err := c.IsQueryable(ctx, "ContentDocumentLink")
	if err != nil {
		return "", err
	}
	docsQueryable, err := c.IsQueryable(ctx, "ContentDocument")
	if err != nil {
		return "", err
	}
	versionsQueryable, err := c.IsQueryable(ctx, "ContentVersion")
	if err != nil {
		return "", err
	}

	if !linksQueryable || !docsQueryable || !versionsQueryable {
		log.Warnf("[%s connector] ContentDocuments, ContentVersions, or ContentDocumentLinks were not queryable, "+
			"so not including in any queries", ConnectorSalesforce)
		c.contentDocumentLinksJoinQuery = ""
		return c.contentDocumentLinksJoinQuery, nil
	}

	// Get queryable fields for ContentDocument
	docsFields, err := c.SelectQueryableFields(ctx, "ContentDocument", []string{
		"Title", "FileExtension", "ContentSize", "Description",
	})
	if err != nil {
		return "", err
	}
	for i, field := range docsFields {
		docsFields[i] = fmt.Sprintf("ContentDocument.%s", field)
	}

	// Get queryable fields for ContentVersion
	versionFields, err := c.SelectQueryableFields(ctx, "ContentVersion", []string{
		"VersionDataUrl", "VersionNumber",
	})
	if err != nil {
		return "", err
	}
	for i, field := range versionFields {
		versionFields[i] = fmt.Sprintf("ContentDocument.LatestPublishedVersion.%s", field)
	}

	// Process file types: remove leading dot and wrap with single quotes
	quotedFiletypes := make([]string, len(TikaSupportedFiletypes))
	for i, filetype := range TikaSupportedFiletypes {
		// Remove leading dot and wrap with single quotes
		quotedFiletypes[i] = fmt.Sprintf("'%s'", filetype[1:])
	}
	whereInClause := strings.Join(quotedFiletypes, ",")
	whereClause := fmt.Sprintf("ContentDocument.FileExtension IN (%s)", whereInClause)

	return NewSalesforceSoqlBuilder("ContentDocumentLinks").
		WithFields(docsFields).
		WithFields(versionFields).
		WithFields([]string{
			"ContentDocument.Id",
			"ContentDocument.LatestPublishedVersion.Id",
			"ContentDocument.CreatedDate",
			"ContentDocument.LastModifiedDate",
			"ContentDocument.LatestPublishedVersion.CreatedDate",
		}).
		WithFields([]string{
			"ContentDocument.Owner.Id",
			"ContentDocument.Owner.Name",
			"ContentDocument.Owner.Email",
		}).
		WithFields([]string{
			"ContentDocument.CreatedBy.Id",
			"ContentDocument.CreatedBy.Name",
			"ContentDocument.CreatedBy.Email",
		}).
		WithWhere(whereClause).
		Build(), nil
}

func (c *SalesforceClient) emailMessageJoinQuery(ctx context.Context) (string, error) {
	desiredFields := []string{"ParentId", "MessageDate", "LastModifiedById", "TextBody", "Subject", "FromName",
		"FromAddress", "ToAddress", "CcAddress", "BccAddress", "Status", "IsDeleted", "FirstOpenedDate",
	}
	queryableFields, err := c.SelectQueryableFields(ctx, "EmailMessage", desiredFields)
	if err != nil {
		return "", fmt.Errorf("failed to get queryable fields for EmailMessage: %w", err)
	}

	return NewSalesforceSoqlBuilder("EmailMessages").
		WithId().
		WithFields(queryableFields).
		WithFields([]string{"CreatedBy.Id", "CreatedBy.Name", "CreatedBy.Email"}).
		WithLimit(500).
		Build(), nil
}

func (c *SalesforceClient) caseCommentsJoinQuery(ctx context.Context) (string, error) {
	desiredFields := []string{"ParentId", "CommentBody", "LastModifiedById"}
	queryableFields, err := c.SelectQueryableFields(ctx, "CaseComment", desiredFields)
	if err != nil {
		return "", fmt.Errorf("failed to get queryable fields for CaseComment: %w", err)
	}

	return NewSalesforceSoqlBuilder("CaseComments").
		WithId().
		WithDefaultMetafields().
		WithFields(queryableFields).
		WithFields([]string{"CreatedBy.Id", "CreatedBy.Name", "CreatedBy.Email"}).
		WithLimit(500).
		Build(), nil
}

func (c *SalesforceClient) caseFeedsQuery(ctx context.Context, caseIds []string) (string, error) {
	desiredFields := []string{"ParentId", "Type", "IsDeleted", "CommentCount", "Title", "Body", "LinkUrl"}
	queryableFields, err := c.SelectQueryableFields(ctx, "CaseFeed", desiredFields)
	if err != nil {
		return "", fmt.Errorf("failed to get queryable fields for CaseFeed: %w", err)
	}

	builder := NewSalesforceSoqlBuilder("CaseFeed").
		WithId().
		WithDefaultMetafields().
		WithFields(queryableFields).
		WithFields([]string{"CreatedBy.Id", "CreatedBy.Name", "CreatedBy.Email"})

	// Build JOIN clause for case feed comments
	joinClause, err := c.caseFeedCommentsJoin(ctx)
	if err != nil {
		log.Warnf("[%s connector] failed to get case feed comments join: %v", ConnectorSalesforce, err)
	} else if joinClause != "" {
		builder.WithJoin(joinClause)
	}

	// Build WHERE clause for ParentId IN (case_ids)
	var whereClause string
	if len(caseIds) > 0 {
		// Wrap each case ID with single quotes and join with commas
		quotedCaseIds := make([]string, len(caseIds))
		for i, caseId := range caseIds {
			quotedCaseIds[i] = fmt.Sprintf("'%s'", caseId)
		}
		whereInClause := strings.Join(quotedCaseIds, ",")
		whereClause = fmt.Sprintf("ParentId IN (%s)", whereInClause)
		builder.WithWhere(whereClause)
	}
	return builder.Build(), nil
}

func (c *SalesforceClient) caseFeedCommentsJoin(ctx context.Context) (string, error) {
	desiredFields := []string{"ParentId", "CreatedDate", "LastEditById", "LastEditDate", "CommentBody",
		"IsDeleted", "StatusParentId",
	}
	queryableFields, err := c.SelectQueryableFields(ctx, "FeedComment", desiredFields)
	if err != nil {
		return "", fmt.Errorf("failed to get queryable fields for FeedComment: %w", err)
	}

	return NewSalesforceSoqlBuilder("FeedComments").
		WithId().
		WithFields(queryableFields).
		WithFields([]string{"CreatedBy.Id", "CreatedBy.Name", "CreatedBy.Email"}).
		WithLimit(500).
		Build(), nil
}
