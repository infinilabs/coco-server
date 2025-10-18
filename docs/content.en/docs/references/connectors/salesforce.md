---
title: "Salesforce"
weight: 60
---

# Salesforce Connector

## Register Salesforce Connector

```shell
curl -XPUT "http://localhost:9000/connector/salesforce?replace=true" -d '{
    "name": "Salesforce Connector",
    "description": "Fetch data from Salesforce with intelligent field caching and query optimization.",
    "icon": "/assets/icons/connector/salesforce/icon.png",
    "category": "crm",
    "tags": [
        "salesforce",
        "crm",
        "api",
        "oauth"
    ],
    "url": "http://coco.rs/connectors/salesforce",
    "assets": {
        "icons": {
            "default": "/assets/icons/connector/salesforce/icon.png",
            "account": "/assets/icons/connector/salesforce/account.png",
            "opportunity": "/assets/icons/connector/salesforce/opportunity.png",
            "contact": "/assets/icons/connector/salesforce/contact.png",
            "lead": "/assets/icons/connector/salesforce/lead.png",
            "campaign": "/assets/icons/connector/salesforce/campaign.png",
            "case": "/assets/icons/connector/salesforce/case.png"
        }
    },
    "processor": {
        "enabled": true,
        "name": "salesforce"
    }
}'
```

> Use `salesforce` as a unique identifier, as it is a builtin connector.

> **Note**: Starting from version **0.4.0**, the Salesforce connector uses a **pipeline-based architecture** for better performance and flexibility. The `processor` configuration is required for the connector to work properly.

## Pipeline Architecture

Starting from version **0.4.0**, the Salesforce connector uses a **pipeline-based architecture** instead of the legacy scheduled task approach. This provides:

- **Better Performance**: Centralized dispatcher manages all connector sync operations
- **Per-Datasource Configuration**: Each datasource can have its own sync interval
- **Enrichment Pipeline Support**: Optional data enrichment pipelines per datasource
- **Resource Efficiency**: Optimized scheduling and resource management

### Pipeline Configuration (coco.yml)

The connector is managed by the centralized dispatcher pipeline:

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

> **Important**: This pipeline configuration replaces the old connector-level config. The dispatcher automatically manages all enabled connectors.

### Connector Configuration

OAuth credentials are configured at the connector level via the management interface or API:

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

### Explanation of Connector Config Parameters

| **Field**      | **Type**  | **Description**                                                                 |
|-----------------|-----------|---------------------------------------------------------------------------------|
| `domain`       | `string`  | Your Salesforce domain (e.g., "mycompany" for mycompany.my.salesforce.com).    |
| `client_id`    | `string`  | OAuth2 client ID from your Salesforce connected app.                           |
| `client_secret`| `string`  | OAuth2 client secret from your Salesforce connected app.                       |
| `processor.enabled` | `boolean` | Enables the pipeline processor (required).                                |
| `processor.name`    | `string`  | Processor name, must be "salesforce" (required).                          |

## Use the Salesforce Connector

The Salesforce Connector allows you to index data from your Salesforce org with intelligent field caching, query optimization, and comprehensive data extraction.

### Features

- **OAuth2 Client Credentials Flow**: Secure server-to-server authentication
- **Intelligent Field Caching**: Caches queryable objects and fields to optimize API calls
- **Query Optimization**: Automatically filters fields to only query accessible ones
- **Standard Objects Support**: Indexes standard Salesforce objects (Account, Opportunity, Contact, Lead, Campaign, Case)
- **Custom Objects Support**: Can index custom objects with the `__c` suffix
- **Case Feeds Integration**: Automatically includes related Case Feeds for comprehensive Case data
- **Content Document Links**: Includes attached files and documents in SOQL queries
- **Relationship Fields**: Supports querying relationship fields like Owner.Id, Owner.Name, Owner.Email
- **Configurable Content Extraction**: Flexible content field mapping for different object types
- **Directory Access Support**: Hierarchical directory structure for browsing Salesforce data by SObject type

### Setup Salesforce Connected App

Before using this connector, you need to create a Salesforce Connected App and configure OAuth2 Client Credentials Flow.

#### 1. Create a Salesforce Connected App

1. Log in to your Salesforce org

{{% load-img "/img/connector/salesforce/salesforce_lightning_setup.png" "Setup" %}}

2. Go to Setup > App Manager

{{% load-img "/img/connector/salesforce/salesforce_lightning_app_manager.png" "App Manager" %}}

3. Click "New External Client App"

4. Fill in the required fields:
    - External Client App Name: "Coco Connector"
    - Api Name: "Coco_Connector"
    - Contact Email: your email

{{% load-img "/img/connector/salesforce/salesforce_lightning_basic_information.png" "Basic Information" %}}

5. Enable OAuth Settings:
    - Callback URL
    - Selected OAuth Scopes:
        - Manage user data via APIs (api)
        - Perform requests at any time (refresh_token, offline_access)
        - Access content resources (content)
      
{{% load-img "/img/connector/salesforce/salesforce_lightning_app_settings.png" "App Settings" %}}

6. **Enable Client Credentials Flow** (Important):
   - Check "Enable Client Credentials Flow"
   - This allows server-to-server authentication without user interaction

{{% load-img "/img/connector/salesforce/salesforce_lightning_flow_enablement.png" "Flow Enablement" %}}

7. Save the connected app

8. Note down the Consumer Key (Client ID) and Consumer Secret (Client Secret)

{{% load-img "/img/connector/salesforce/salesforce_lightning_key_secret.png" "Consumer Key and Secret" %}}

#### 2. Enable Client Credentials User (if needed)

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

### 3. Enable Client Credentials Flow

1. **Enable "Client Credentials Flow"** for this user:
   - Go to Setup > App Manager > Your Connected App
   - In the "Run As (Username)" section, assign the user
   - Ensure the user is active and has API access 

{{% load-img "/img/connector/salesforce/salesforce_lightning_client_credentials.png" "Client Credentials Flow" %}}


### Access Connector Settings

1. Navigate to the **Data Sources** section in your Coco dashboard
2. Find the Salesforce data source you want to configure
3. Click the **Edit** button (pencil icon) next to the data source
4. This will open the configuration page where you can:
   - Modify which Salesforce objects to sync
   - Enable or disable custom object synchronization
   - Update sync settings and filters

> **⚠️ Important**: Before you can use the Salesforce connector, you must configure the following required parameters:
> - `domain`: Your Salesforce domain (e.g., "mycompany" for mycompany.my.salesforce.com)
> - `client_id`: OAuth2 client ID from your Salesforce connected app
> - `client_secret`: OAuth2 client secret from your Salesforce connected app
> 
> These credentials are obtained from the Salesforce Connected App you created in the previous steps.

{{% load-img "/img/connector/salesforce/salesforce_connector.png" "Configure Salesforce OAuth" %}}

### Datasource Configuration

Each datasource has its own sync configuration and object selection:

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '{
    "name": "My Salesforce Data",
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
}'
```

### Datasource Config Parameters

| **Field**                    | **Type**   | **Description**                                                                                |
|------------------------------|------------|------------------------------------------------------------------------------------------------|
| `standard_objects_to_sync`   | `array`    | List of standard objects to sync (default: all standard objects).                             |
| `sync_custom_objects`        | `boolean`  | Whether to sync custom objects (default: false).                                              |
| `custom_objects_to_sync`     | `array`    | List of custom objects to sync (use "*" for all).                                             |
| `sync.enabled`               | `boolean`  | Enable/disable syncing for this datasource.                                                   |
| `sync.interval`              | `string`   | Sync interval for this datasource (e.g., "5m", "1h", "30s").                                  |

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

## Supported Config Parameters for Salesforce Connector

Below are the configuration parameters supported by the Salesforce Connector:

| **Field**                    | **Type**   | **Description**                                                                                |
|------------------------------|------------|------------------------------------------------------------------------------------------------|
| `standard_objects_to_sync`   | `array`    | List of standard objects to sync (default: all standard objects).                             |
| `sync_custom_objects`        | `boolean`  | Whether to sync custom objects (default: false).                                              |
| `custom_objects_to_sync`     | `array`    | List of custom objects to sync (use "*" for all).                                             |

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

## Notes

- The connector uses OAuth2 Client Credentials Flow for server-to-server authentication
- Field caching significantly reduces API calls and improves performance
- Case Feeds are automatically included for Case objects when available
- Custom objects must have the `__c` suffix to be recognized
- The connector supports incremental synchronization based on LastModifiedDate
