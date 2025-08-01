---
weight: 80
title: 'Release Notes'
---

# Release Notes

Information about release notes of Coco Server is provided here.

## Latest (In development)  
### ❌ Breaking changes  
### 🚀 Features  
### 🐛 Bug fix  
- fix: correct assistant update logic
- fix: generate unique icon key to prevent accidental deletion of all icons
### ✈️ Improvements  
- chore: remove unused websocket api #443

## 0.7.0 (2025-07-25)
### ❌ Breaking changes  
- refactor: refactoring mappings #394

### 🚀 Features  
- feat: add http_streaming based chat api #336
- feat: add file upload config #349
- feat: support attachments in chat message #355
- feat: log llm request for debugging purpose
- feat: rss connector #149
- feat: support default model reasoning config on initialization
- feat: local FS Connector #284
- feat: s3 Connector #68
- feat: confluence wiki connector #31

### 🐛 Bug fix  
- fix: query parameter "filter" is not working
- fix: pagination in the list is not working
- fix: local icons fail to display without network
- fix: incorrect status display in llm provider list #364
- fix: chat api with attachments #368
- fix: prevent nil exception during LLM intent parsing on error #375
- fix: deleting multiple URL input boxes does not work #393
- fix: fix missing update of local model providers after enable operation
- fix: ensure datasource is used during RAG processing #396
- fix: incorrect picking prompt template #411
- fix: prevent loss of reply message when user cancels ongoing reply
- fix: make the first chat message cancelable


### ✈️ Improvements  
- refactor: refactoring user id #337
- chore: skip empty response chunks #338
- refactor: refactoring query #340
- chore: mask more senstive search results #343
- refactor: refactoring attachement api #350
- chore: add upload settings to assistant #352
- refactor: refactoring ORM and security interface
- chore: remove session_id check in attachment upload api #357
- chore: add formatUrl to searchbox #360
- chore: add fullscreen to integration #373
- chore: ignore invalid connector #379
- chore: skip invalid mcp server #380
- improve: hide delete button for built-in assistants and providers
- chore: handle default value for prompt template
- chore: set button preview to disabled if integration is disabled #392
- chore: manual flush the first line #396

## 0.6.0 (2025-06-29)
### ❌ Breaking changes  
### 🚀 Features  
### 🐛 Bug fix  
- fix: remove manually_renamed_title from assistant search

### ✈️ Improvements  

## 0.5.0 (2025-06-06)

### ❌ Breaking changes

### 🚀 Features

- feat: allow converting icon to base64 #261
- feat: implement ask api for assistant
- feat: add placeholder to chat settings
- feat: return number of assistants in provider info API
- feat: add assistant to search results #274
- feat: add built-in AI assistant `AI Overview`
- feat: add throttle filter wth request fingerprint #294
- feat: multi-user login support

### 🐛 Bug fix

- fix: add missing cors feature flags to settings api
- fix: incorrect datasource icon #265
- fix: handle empty URL values in HugoSite-type datasource
- fix: datasource & MCP selection problem #267
- fix: resolve compatibility issue with crypto.randomUUID when creating model provider
- fix: start page configuration of integration is not working
- fix: unchecked datasource/mcp_server not working

### ✈️ Improvements

- chore: clean up unused LLM settings code
- chore: sort chat history by created
- chore: add enabled by default params to assistant edit
- chore: password supports more special characters
- refactor: refactoring chat api #273
- chore: add placeholder, category and tags to AI Assistant
- chore: ignore empty chunk_message #288
- refactor: refactoring search #323

## 0.4.0 (2025-04-27)

### Breaking changes

### Features

- Add chat session management API
- Add support for font icons (#183)
- Add support for AI assistant CURD management
- Add support for model provider CURD management
- Add version and license

### Bug fix

- Fix personal token was not well-supported for Yuque connector
- Fix incorrect content-type header for wrapper
- Fix default login url can't be changed afterward

### Improvements

- Set built-in connector icons as read-only
- Support setting icon and placeholder of integration
- Enhance UI for searchbox
- Refactoring security plugin #199
- Make searchbox's theme styles follows the system if searchbox's theme is set to `auto`
- Support setting suggested topics of integration
- Skip handle wrapper for disabled widget
- When creating a new Google Drive datasource, guide the user to configure the required settings if they are missing
- Default to use go modules
- Support user-provided icon URL in icon component
- Update default query template

## 0.3.0 (2025-03-31)

### Breaking changes

### Features

- Add support for Connector CRUD management (#147)
- Control the searchability of related documents based on the data source's enabled status. (#147)
- Allow user pass websocket session id via request header #148
- Add integration management API
- Add searchbox widget for easy of embedding to website
- Add support for integration CRUD management and CORS configuration (#153)
- Add api to delete attachment
- Add dynamic js wrapper for widget
- Parse document icon at the server side
- Add suggest topcs to widget integration
- Add support to filter senstive fields

### Bug fix

- Fixed provider info version (#144)
- Fixed an issue where keyword search filtering for datasource was not working as expected (#147)
- Fixed to remove uncheck datasource condition in must conditions

### Improvements

## 0.2.2 (2025-03-14)

### Breaking changes

### Features

- Add support for API token CRUD management (#132)
- Add shortcut API to create doc in datasource
- Add attachment API to management uploaded files in chat session

### Bug fix

- Fixed fatal error: concurrent map writes #125

### Improvements

- Enhance UI for Adding a New Data Source (#126)
- Add option login flag to logout api
- Catch error in background message processing task
- Optimize RAG tasks
- Throw error on invalid user during WebSocket connection

## 0.2.0 (2025-03-07)

### 🚀 Features

- Add default index template and schema to document
- Implement docoument serach api
- Implement suggest api
- Support cancel inflight background job
- Add google drive connector
- Incremental indexing google drive files, connect via url
- Ignore empty query (#35)
- Add new field to push messages (#34)
- Add reset api to google_drive's connector (#36)
- Add yuque connector #41
- Allow to skip invalid token for yuque connector
- Add hugo site connector (#51)
- Add datasource and connector
- Add notion connector (#63)
- Add document enrichment processor
- Init support for RAG
- Add web #77
- Add a simple security feature to Coco Server (#79)
- Init commit for Datasource management UI (#81)
- _(datasource)_ Support CRUD management (#82)
- Add guide, login, home, and settings (#83)
- Add field `SyncEnabled` to control datasource synchronization (#103)
- Add google drive connector settings (#109)
- Support toggling synchronization in datasource list (#112)
- Add LLM config (intent_analysis_model, picking_doc_model, answe… (#114)

### 🐛 Bug Fixes

- Update header key to avoid using underscores (#48)
- Init the payload
- Adjust locales (#85)
- Adjust endpoint validation (#96)
- Adjust styles of guide (#97)
- Adjust locales of llm (#100)
- Adjust loading
- Update settings of yuque datasource not work (#104)
- Override theme settings (#105)
- Avoid panic when Google Drive credential config is missing (#106)
- Empty response (#121)
- Redirect to the login page when the token expires (#124)
- Fatal error: concurrent map writes

### 🚜 Refactor

- Split metadata and payload
- Refactoring datasource
- Refactoring icon management
- Refactoring hugo_site connector to support mutlti datasource (#56)
- Refactoring google_drive connector
- Refactoring query and suggest
- Refactoring connectors
- Refactoring connectors
- Refactoring google_drive connector, support token_refresh, … (#61)
- Refactoring yuque connector (#62)
- Refactoring static assets (#65)
- Refactoring default config
- Refactoring search api
- Refactoring rag based chat (#94)

### 📚 Documentation

- Update search document
- Init docs (#47)
- Add connectors
- Fix images
- Update docker install guide (#118)
- Update install docs (#119)
- Typo update docker install guide (#120)
- Upgrade to 0.2.1 (#122)
- Add outputs json (#123)

### ⚙️ Miscellaneous Tasks

- Update default templates
- Update README
- Add tips about websocket (#16)
- Fix typo (#17)
- Naming style
- Update readme
- Remove single quotes from example
- Update import reflect to refactoring
- Update logging level
- Check oauth config
- Update github PR template
- Update default page size to 10
- Update license
- Update Makefile
- Ignore empty query
- Skip setup early
- Disable metadata for indexing
- Update pull request template (#39)
- Update terminal header (#43)
- Update missing import (#44)
- Add subcategory
- Update tips
- Update default port to 9000
- Builtin connectors should use builtin id
- Update code sample
- Update code format (#55)
- Update default config
- Update docs
- Remove unused code
- Fix icon link
- Update api docs (#60)
- Remove redundant last category if it matches the document title (#64)
- Update yuque connector
- Add missing provider
- Remove osv-scanner.yml
- Update docs
- Remove langchaingo from source
- Update docs (#80)
- Fix build web (#84)
- Update settings (#86)
- Add icon to datesource list (#95)
- Minor fix (#98)
- Remove basic auth doc (#99)
- Update locales of data source (#101)
- Adjust locales in data source (#107)
- Adjust styles of loading (#108)
- Echo message before pick docs (#110)
- Expose models to config (#111)
- Update to support ollama (#113)
- Update default proxy enabled to `false` (#115)
- Default banner (#116)
- Show token config when LLM type is deepseek (#117)

## 0.1.0 (2025-02-16)

### Features

- Indexing API
- Search API
- Suggest API
- Assistant Chat API
- Google Drive Connector
- Yuque Connector
- Notion Connector
- RAG based Chat
- Basic security

### Breaking changes

### Bug fix

### Improvements

- Update header key to avoid using underscores #48
