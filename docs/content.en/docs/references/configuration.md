---
title: "Configuration"
weight: 5
---

# Configuration Reference

Coco Server is configured through a YAML file (`coco.yml`). This reference documents all available configuration options with detailed examples.

## Environment Variables

Environment variables can be defined in the `env` section and referenced elsewhere in the configuration using `$[[env.VAR_NAME]]`.

```yaml
env:
  ES_ENDPOINT: https://localhost:9200
  ES_USERNAME: admin
  ES_PASSWORD: $[[keystore.ES_PASSWORD]]
  WEB_BINDING: 0.0.0.0:9000
  API_BINDING: 0.0.0.0:2900
```

> Sensitive values can be stored securely using the keystore: `./bin/coco keystore add ES_PASSWORD`

## Coco Server Settings

Core server settings are defined under the `coco` section.

```yaml
coco:
  server:
    public: false
    name: "My Coco Server"
    encode_icon_to_base64: false
    minimal_client_version:
      number: "0.3"
    provider:
      name: "INFINI Labs"
      description: "Coco AI Server - Search, Connect, Collaborate."
      icon: "https://coco.rs/favicon.ico"
      website: "https://coco.rs/"
      eula: "https://coco.rs/#/terms"
      privacy_policy: "https://coco.rs/privacy"
      banner: "https://coco.rs/svg/connect.svg"
    store:
      endpoint: "https://coco.infini.cloud"
      local: false
```

| **Field**                              | **Type**  | **Default**   | **Description**                                                                |
|----------------------------------------|-----------|---------------|--------------------------------------------------------------------------------|
| `coco.server.public`                   | `boolean` | `false`       | Whether the server is publicly accessible.                                     |
| `coco.server.name`                     | `string`  | `""`          | Display name for the Coco Server instance.                                     |
| `coco.server.encode_icon_to_base64`    | `boolean` | `false`       | Encode connector icons to base64. Enable if using self-signed certificates.    |
| `coco.server.minimal_client_version.number` | `string` | `""`      | Minimum required client version number.                                        |
| `coco.server.provider.name`            | `string`  | `""`          | Organization name shown in the UI.                                             |
| `coco.server.provider.description`     | `string`  | `""`          | Server description text.                                                       |
| `coco.server.provider.icon`            | `string`  | `""`          | URL to provider icon.                                                          |
| `coco.server.provider.website`         | `string`  | `""`          | Provider website URL.                                                          |
| `coco.server.provider.eula`            | `string`  | `""`          | End User License Agreement URL.                                                |
| `coco.server.provider.privacy_policy`  | `string`  | `""`          | Privacy policy URL.                                                            |
| `coco.server.provider.banner`          | `string`  | `""`          | Banner image URL.                                                              |
| `coco.server.store.endpoint`           | `string`  | `""`          | Coco extension store endpoint.                                                 |
| `coco.server.store.local`              | `boolean` | `false`       | Use local store service.                                                       |

### Managed Mode

For managed deployments (e.g., multi-tenant), enable managed mode with SSO authentication:

```yaml
coco:
  server:
    endpoint: "http://localhost:9001/"
    auth_provider:
      sso:
        url: "http://localhost:9001/#/login"
    provider:
      auth_provider:
        sso:
          url: "/sso/login/cloud?provider=coco-cloud&product=coco"
```

## Elasticsearch Connection

Coco Server uses Elasticsearch (or Easysearch) as its primary data store.

```yaml
elasticsearch:
  - name: prod
    enabled: true
    endpoints:
      - $[[env.ES_ENDPOINT]]
    discovery:
      enabled: false
    basic_auth:
      username: $[[env.ES_USERNAME]]
      password: $[[env.ES_PASSWORD]]
```

| **Field**               | **Type**        | **Description**                                              |
|-------------------------|-----------------|--------------------------------------------------------------|
| `name`                  | `string`        | Name identifier for this Elasticsearch cluster.              |
| `enabled`               | `boolean`       | Enable or disable this connection.                           |
| `endpoints`             | `array[string]` | List of Elasticsearch endpoint URLs.                         |
| `discovery.enabled`     | `boolean`       | Enable node discovery for the cluster.                       |
| `basic_auth.username`   | `string`        | Username for basic authentication.                           |
| `basic_auth.password`   | `string`        | Password for basic authentication.                           |

## API Server

Internal API server configuration (typically used for inter-service communication).

```yaml
api:
  enabled: true
  websocket:
    enabled: false
  network:
    binding: $[[env.API_BINDING]]
```

## Web Server

The web server serves the UI and public-facing API endpoints.

```yaml
web:
  enabled: true
  embedding_api: false
  network:
    binding: $[[env.WEB_BINDING]]
  tls:
    enabled: false
    skip_insecure_verify: true
    default_domain: "localhost:2900"
    auto_issue:
      enabled: false
      email: "hello@infinilabs.com"
      include_default_domain: true
      domains:
        - "www.coco.rs"
  security:
    enabled: true
    managed: false
    authentication:
      native:
        enabled: true
      oauth:
        cloud:
          enabled: true
          client_secret: "999999"
          authorize_url: "http://localhost:9000/oauth/authorize"
          token_url: "http://localhost:9000/oauth/token"
          profile_url: "http://localhost:9000/account/profile"
          redirect_url: "http://localhost:9001/sso/callback/cloud"
          success_page: "/#/home"
          failed_page: "/#/login"
          scopes: ["openid", "email", "profile"]
```

| **Field**                       | **Type**  | **Description**                                            |
|---------------------------------|-----------|------------------------------------------------------------|
| `web.enabled`                   | `boolean` | Enable the web server.                                     |
| `web.embedding_api`             | `boolean` | Enable embedding API endpoint.                             |
| `web.network.binding`           | `string`  | Network address and port to bind to.                       |
| `web.tls.enabled`               | `boolean` | Enable TLS/HTTPS.                                          |
| `web.tls.skip_insecure_verify`  | `boolean` | Skip TLS certificate verification.                         |
| `web.tls.auto_issue.enabled`    | `boolean` | Enable automatic TLS certificate issuance (Let's Encrypt). |
| `web.security.enabled`          | `boolean` | Enable security features.                                  |
| `web.security.managed`          | `boolean` | Enable managed mode (remote managed instance).             |

## Pipeline Configuration

Pipelines define background processing jobs for document enrichment, indexing, and connector dispatching.

### Document Enrichment Pipeline

```yaml
pipeline:
  - name: enrich_documents
    auto_start: false
    keep_running: true
    processor:
      - consumer:
          auto_commit_offset: true
          queue_selector:
            keys:
              - indexing_documents
          consumer:
            group: enriched_documents
            fetch_max_messages: 10
          processor:
            - file_type_detection: {}
            - file_extraction:
                tika_endpoint: "http://127.0.0.1:9998"
                tika_timeout_in_seconds: 360
                chunk_size: 7000
                vision_model_provider: qianwen
                vision_model: qwen-vl-max
            - document_summarization:
                model_provider: deepseek
                model: deepseek-chat
                model_context_length: 128000
                ai_insights_max_length: 500
            - extract_tags:
                model_provider: deepseek
                model: deepseek-chat
                model_context_length: 128000
            - document_embedding:
                model_provider: qianwen
                model: text-embedding-v4
                embedding_dimension: 1024
                chunk_size: 7000
                output_queue:
                  name: "enriched_documents"
```

| **Processor**            | **Description**                                                                              |
|--------------------------|----------------------------------------------------------------------------------------------|
| `file_type_detection`    | Detects the file type of uploaded documents.                                                 |
| `file_extraction`        | Extracts text content from files using Apache Tika.                                          |
| `document_summarization` | Generates AI-powered summaries of document content.                                          |
| `extract_tags`           | Extracts relevant tags from documents using AI.                                              |
| `document_embedding`     | Generates vector embeddings for semantic search.                                             |

### Document Merge Pipeline

```yaml
pipeline:
  - name: merge_documents
    auto_start: true
    keep_running: true
    processor:
      - indexing_merge:
          input_queue: "indexing_documents"
          idle_timeout_in_seconds: 1
          elasticsearch: "prod"
          index_name: "coco_document-v2"
          key_field: "id"
          output_queue:
            name: "merged_documents"
            label:
              tag: "merged"
          worker_size: 1
          bulk_size_in_kb: 10240
```

### Document Ingestion Pipeline

```yaml
pipeline:
  - name: ingest_documents
    auto_start: true
    keep_running: true
    processor:
      - bulk_indexing:
          bulk:
            compress: true
            batch_size_in_mb: 10
            batch_size_in_docs: 10240
          consumer:
            fetch_max_messages: 100
          queues:
            type: indexing_merge
            tag: "merged"
```

### Connector Dispatcher

The connector dispatcher manages scheduled data synchronization from configured datasources.

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

| **Field**                            | **Type**  | **Description**                                           |
|--------------------------------------|-----------|-----------------------------------------------------------|
| `auto_start`                         | `boolean` | Automatically start the pipeline on server startup.       |
| `keep_running`                       | `boolean` | Restart the pipeline if it stops.                         |
| `singleton`                          | `boolean` | Only one instance of this pipeline can run at a time.     |
| `retry_delay_in_ms`                  | `int`     | Delay in milliseconds before retrying on failure.         |
| `max_running_timeout_in_seconds`     | `int`     | Maximum time a connector sync can run before timeout.     |

## HTTP Client / Proxy Settings

Configure proxy settings for outbound HTTP requests (e.g., to external APIs and connectors).

```yaml
http_client:
  default:
    proxy:
      enabled: false
      default_config:
        http_proxy: http://127.0.0.1:7890
        socket5_proxy: socks5://127.0.0.1:7890
      override_system_proxy_env: true
      permitted:
        - "google.com"
      denied:
        - "localhost"
        - "infinilabs.com"
      domains:
        "github.com":
          http_proxy: http://127.0.0.1:7890
          socket5_proxy: socks5://127.0.0.1:7890
```

| **Field**                     | **Type**        | **Description**                                                      |
|-------------------------------|-----------------|----------------------------------------------------------------------|
| `proxy.enabled`               | `boolean`       | Enable proxy for outbound HTTP requests.                             |
| `proxy.default_config`        | `object`        | Default proxy configuration for all requests.                        |
| `proxy.override_system_proxy_env` | `boolean`   | Override system proxy environment variables.                         |
| `proxy.permitted`             | `array[string]` | List of domains allowed to use the proxy.                            |
| `proxy.denied`                | `array[string]` | List of domains that should bypass the proxy.                        |
| `proxy.domains`               | `object`        | Per-domain proxy configuration overrides.                            |

## Minimal Configuration Example

A minimal `coco.yml` to get started:

```yaml
env:
  ES_ENDPOINT: https://localhost:9200
  ES_USERNAME: admin
  ES_PASSWORD: changeme
  WEB_BINDING: 0.0.0.0:9000
  API_BINDING: 0.0.0.0:2900

coco:
  server:
    name: "My Coco Server"

elasticsearch:
  - name: prod
    enabled: true
    endpoints:
      - $[[env.ES_ENDPOINT]]
    basic_auth:
      username: $[[env.ES_USERNAME]]
      password: $[[env.ES_PASSWORD]]

web:
  enabled: true
  network:
    binding: $[[env.WEB_BINDING]]
  security:
    enabled: true

api:
  enabled: true
  network:
    binding: $[[env.API_BINDING]]
```

## Environment Variable Overrides

You can override configuration values using environment variables when starting the server:

```bash
ES_PASSWORD=mypassword WEB_BINDING=0.0.0.0:8080 ./bin/coco
```
