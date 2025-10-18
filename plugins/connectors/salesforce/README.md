# Salesforce Connector

This connector integrates with Salesforce to index and search data from your Salesforce org with intelligent field caching and query optimization.

## Features

- **OAuth2 Client Credentials Flow**: Secure server-to-server authentication
- **Intelligent Field Caching**: Caches queryable objects and fields to optimize API calls
- **Query Optimization**: Automatically filters fields to only query accessible ones
- **Standard Objects Support**: Indexes standard Salesforce objects (Account, Opportunity, Contact, Lead, Campaign, Case)
- **Custom Objects Support**: Can index custom objects with the `__c` suffix
- **Case Feeds Integration**: Automatically includes related Case Feeds for comprehensive Case data
- **Content Document Links**: Includes attached files and documents in SOQL queries
- **Relationship Fields**: Supports querying relationship fields like Owner.Id, Owner.Name, Owner.Email
- **Incremental Sync**: Supports incremental synchronization based on LastModifiedDate
- **Document Level Security**: Optional document-level security based on Salesforce permissions
- **Content Extraction**: Supports text extraction from Salesforce content documents
- **Error Prevention**: Validates object queryability before attempting queries
- **Configurable Content Extraction**: Flexible content field mapping for different object types
- **Directory Access Support**: Hierarchical directory structure for browsing Salesforce data by SObject type

## Architecture

The Salesforce connector follows the **pipeline-based architecture** introduced in the connector refactoring. It implements the `ConnectorProcessorBase` interface and is managed by the centralized `connector_dispatcher` pipeline.

### Pipeline Integration

- **Processor Registration**: Registered as a pipeline processor in `init()`
- **Dispatcher Managed**: Sync intervals and scheduling handled by dispatcher
- **Per-Datasource Config**: Each datasource has its own sync interval and configuration
- **Enrichment Pipeline Support**: Supports optional enrichment pipelines per datasource

## Configuration

The connector uses connector-level OAuth configuration and datasource-level sync settings.

### Required Parameters

#### Connector-level Configuration
OAuth credentials are configured at the connector level and shared across all datasources:
- `domain`: Your Salesforce domain (e.g., "mycompany" for mycompany.my.salesforce.com)
- `client_id`: OAuth2 client ID from your Salesforce connected app
- `client_secret`: OAuth2 client secret from your Salesforce connected app

#### Datasource-level Configuration
Each datasource can have its own configuration:
- `standard_objects_to_sync`: List of standard objects to sync (default: all standard objects)
- `sync_custom_objects`: Whether to sync custom objects (default: false)
- `custom_objects_to_sync`: List of custom objects to sync (use "*" for all)
- `sync.enabled`: Enable/disable syncing for this datasource
- `sync.interval`: Sync interval for this datasource (e.g., "5m", "1h")

### Example Configuration

#### Connector Registration
```json
{
  "id": "salesforce",
  "name": "Salesforce Connector",
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "salesforce"
  },
  "config": {
    "domain": "mycompany",
    "client_id": "your_client_id_here",
    "client_secret": "your_client_secret_here"
  }
}
```

#### Datasource Configuration
```json
{
  "name": "My Salesforce Org",
  "type": "connector",
  "enabled": true,
  "connector": {
    "id": "salesforce",
    "config": {
      "standard_objects_to_sync": ["Account", "Opportunity", "Contact", "Lead", "Campaign", "Case"],
      "sync_custom_objects": true,
      "custom_objects_to_sync": ["CustomObject__c"]
    }
  },
  "sync": {
    "enabled": true,
    "interval": "5m"
  }
}
```

#### Pipeline Configuration (coco.yml)
The connector is managed by the centralized dispatcher:
```yaml
pipeline:
  - name: connector_dispatcher
    auto_start: true
    keep_running: true
    singleton: true
    retry_delay_in_ms: 10000
    processor:
      - connector_dispatcher:
          max_running_timeout_in_seconds: 1200
```

## Setup Instructions

### 1. Create a Salesforce Connected App

1. Log in to your Salesforce org
2. Go to Setup > App Manager
3. Click "New Connected App"
4. Fill in the required fields:
   - Connected App Name: "Coco Connector"
   - API Name: "Coco_Connector"
   - Contact Email: your email
5. Enable OAuth Settings:
   - Selected OAuth Scopes:
     - Access and manage your data (api)
     - Perform requests on your behalf at any time (refresh_token, offline_access)
6. **Enable Client Credentials Flow** (Important):
   - Check "Enable Client Credentials Flow"
   - This allows server-to-server authentication without user interaction
7. Save the connected app
8. Note down the Consumer Key (Client ID) and Consumer Secret (Client Secret)

### 2. Enable Client Credentials User

1. Go to Setup > Users > Permission Sets
2. Create a new Permission Set or use an existing one
3. Add the following permissions:
   - API Enabled
   - View All Data (if needed)
   - Modify All Data (if needed)
4. Go to Setup > Users > Users
5. Find the user you want to use for the connector
6. Click "Edit" next to the user
7. Go to "Permission Set Assignments"
8. Assign the permission set created above
9. **Enable "Client Credentials Flow"** for this user:
   - Go to Setup > App Manager > Your Connected App
   - In the "Client Credentials Flow" section, assign the user
   - Ensure the user is active and has API access

### 3. Configure the Connector

1. Configure the connector with OAuth credentials at connector level
2. Create datasources with specific sync settings
3. The connector_dispatcher pipeline will automatically manage syncing

#### Connector Configuration

Update the connector configuration via API or management console:
```json
{
  "id": "salesforce",
  "config": {
    "domain": "mycompany",
    "client_id": "your_client_id_here",
    "client_secret": "your_client_secret_here"
  }
}
```

#### Create Datasource

Create a datasource to start syncing:
```json
{
  "name": "My Salesforce Org",
  "type": "connector",
  "connector": {
    "id": "salesforce",
    "config": {
      "standard_objects_to_sync": ["Account", "Contact", "Lead", "Opportunity", "Case"],
      "sync_custom_objects": false
    }
  },
  "sync": {
    "enabled": true,
    "interval": "5m"
  }
}
```

### 4. OAuth2 Client Credentials Flow

The connector uses OAuth2 Client Credentials flow for server-to-server authentication:

- **Automated**: No user interaction required
- **Secure**: Uses client_id and client_secret for authentication
- **Efficient**: Optimized for bulk data synchronization
- **Cached**: Intelligent field caching reduces API calls


## Supported Objects

### Standard Objects

- **Account**: Company information, billing addresses, contacts, website, type
- **Opportunity**: Sales opportunities, stages, amounts, related opportunities
- **Contact**: Individual contacts, email, phone, titles, owner information
- **Lead**: Potential customers, lead sources, conversion status, company info
- **Campaign**: Marketing campaigns, status, dates, campaign type
- **Case**: Support cases, status, descriptions, case feeds, comments, and related activities

### Custom Objects

- Any custom object with `__c` suffix
- Supports all custom fields and relationships

## Data Mapping

The connector maps Salesforce data to common document fields:

- `Id` → Document ID
- `Name` → Document Title
- `Description` → Document Content (with object-specific fields)
- `CreatedDate` → Document Created Date
- `LastModifiedDate` → Document Updated Date
- `Owner` → Document Owner (Id, Name, Email)
- Object type → Document Type and Category
- `Feeds` → Case Feeds (for Case objects only)
- `Id` + `instanceUrl` → Document URL (direct link to Salesforce record)

### Directory Structure

The connector creates a hierarchical directory structure for easy browsing:

```
Standard Objects/
├── Account/
├── Contact/
├── Lead/
├── Opportunity/
├── Case/
└── Campaign/

Custom Objects/
├── CustomObject1__c/
└── CustomObject2__c/
```

- **First Level**: SObject type groups (Standard Objects, Custom Objects)
- **Second Level**: Individual SObject types (Account, Contact, etc.)
- **Third Level**: Individual records within each SObject type

### Content Extraction

The connector intelligently extracts content based on object type:

- **Account**: Description, Website, Type, Billing Address
- **Opportunity**: Description, Stage Name
- **Contact**: Description, Email, Phone, Title
- **Lead**: Description, Company, Email, Phone, Status
- **Campaign**: Description, Type, Status, Active status
- **Case**: Description, Case Number, Status, Open/Closed status, Feeds

## Advanced Features

### Intelligent Field Caching

The connector implements intelligent field caching to optimize API performance:

- **Object Caching**: Caches queryable SObjects to avoid repeated API calls
- **Field Caching**: Caches queryable fields for each object type
- **Smart Filtering**: Automatically filters fields to only query accessible ones
- **Error Prevention**: Validates object queryability before attempting queries

#### Caching Methods

- `GetQueryableSObjects()`: Returns cached list of queryable objects
- `GetQueryableSObjectFields()`: Returns cached fields for specific objects
- `IsQueryable()`: Checks if an object is queryable
- `SelectQueryableFields()`: Selects only queryable fields for an object

### Query Optimization

The connector automatically optimizes SOQL queries:

- **Field Validation**: Only queries fields that exist and are accessible
- **Object Validation**: Checks object queryability before querying
- **Dynamic Field Selection**: Adapts queries based on available fields
- **Relationship Fields**: Automatically includes Owner and CreatedBy relationship fields
- **Content Document Links**: Includes attached files and documents in queries
- **Error Reduction**: Prevents common query errors

### Case Feeds Integration

For Case objects, the connector automatically includes related Feeds:

- **Automatic Detection**: Checks if CaseFeed is queryable
- **Batch Processing**: Processes Case Feeds in batches of 800 for performance
- **Feed Grouping**: Groups feeds by ParentId (Case ID)
- **Comprehensive Data**: Includes feed comments, activities, and related content
- **Performance Optimized**: Reduces API calls through intelligent batching

### SOQL Query Builder

The connector uses a fluent SOQL query builder for complex queries:

- **Fluent API**: Chainable methods for building queries
- **Field Management**: Automatic field deduplication and ordering
- **Join Support**: Built-in support for subqueries and joins
- **Conditional Logic**: Support for WHERE, ORDER BY, and LIMIT clauses

### Directory Access

The connector supports hierarchical directory access for easy data browsing:

- **Automatic Directory Creation**: Creates directory structure based on SObject types
- **Hierarchical Navigation**: Browse data by SObject type groups and individual types
- **Metadata Support**: Each directory includes metadata about SObject types
- **Path-based Access**: Use directory paths to access specific SObject data
- **Standard vs Custom Objects**: Clear separation between standard and custom objects

#### Directory Features

- **Root Level**: No root directory - direct access to SObject type groups
- **Type Grouping**: Standard Objects and Custom Objects are grouped separately
- **Individual SObject Types**: Each SObject type gets its own directory
- **Record Organization**: Individual records are organized under their respective SObject type directories
- **Metadata**: Directories include SObject type information and access metadata


## Troubleshooting

### Common Issues

1. **"no client credentials user enabled" Error**:
   - **Cause**: Client Credentials Flow is not enabled or no user is assigned for client credentials
   - **Solution**: 
     - Go to Setup > App Manager > Your Connected App
     - Check "Enable Client Credentials Flow"
     - Assign a user to the Client Credentials Flow
     - Ensure the user has API access and necessary permissions

2. **"invalid_client" Error**:
   - **Cause**: Incorrect client_id or client_secret
   - **Solution**: Verify your Consumer Key (Client ID) and Consumer Secret (Client Secret) in the Connected App

3. **"invalid_grant" Error**:
   - **Cause**: Authentication grant is invalid
   - **Solution**: Check that the user assigned to Client Credentials Flow is active and has proper permissions

4. **Authentication Failed**: Check your client_id and client_secret
5. **Permission Denied**: Ensure your connected app has the necessary OAuth scopes
6. **Object Not Found**: Verify the object name and that it exists in your org
7. **Field Not Accessible**: Check field-level security settings in Salesforce

### Logs

Check the coco-server logs for detailed error messages:

```bash
tail -f logs/coco.log | grep "salesforce connector"
```

## API Reference

### SalesforceClient Methods

#### Core Methods
- `Authenticate(ctx context.Context) error`: Authenticate with Salesforce using OAuth2 client credentials
- `QueryObject(ctx context.Context, objectType string) ([]map[string]interface{}, error)`: Query a specific object with field caching
- `QueryWithSOQL(ctx context.Context, query string) ([]map[string]interface{}, error)`: Execute custom SOQL query
- `executeQuery(ctx context.Context, query string, useAuthenticatedRequest bool) ([]map[string]interface{}, error)`: Common query execution with pagination

#### Field Caching Methods
- `GetQueryableSObjects(ctx context.Context) ([]string, error)`: Get cached list of queryable objects
- `GetQueryableSObjectFields(ctx context.Context, relevantObjects []string, relevantSObjectFields []string) (map[string][]string, error)`: Get cached fields for objects
- `IsQueryable(ctx context.Context, sobject string) (bool, error)`: Check if an object is queryable
- `SelectQueryableFields(ctx context.Context, sobject string, fields []string) ([]string, error)`: Select only queryable fields

#### Query Building
- `buildSOQLQuery(ctx context.Context, objectType string) (string, error)`: Build optimized SOQL query with field caching
- `caseFeedsQuery(caseIds []string) string`: Build Case Feeds query with WHERE clause
- `contentDocumentLinksJoin(ctx context.Context) (string, error)`: Build content document links subquery

#### Case Feeds Methods
- `processCaseWithFeeds(ctx context.Context, client *SalesforceClient, datasource *common.DataSource)`: Process Case objects with Feeds
- `getCaseFeedsByCaseId(ctx context.Context, client *SalesforceClient, caseIds []string) map[string][]map[string]interface{}`: Get Case Feeds grouped by Case ID

#### SOQL Builder
- `NewSalesforceSoqlBuilder(table string) *SalesforceSoqlBuilder`: Create new SOQL builder
- `WithId()`, `WithDefaultMetafields()`, `WithFields()`, `WithWhere()`, `WithOrderBy()`, `WithLimit()`, `WithJoin()`: Fluent API methods


## License

Copyright © INFINI LTD. All rights reserved.
