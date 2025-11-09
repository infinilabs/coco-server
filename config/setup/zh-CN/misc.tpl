POST $[[SETUP_INDEX_PREFIX]]model-integration$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/full-screen-widget-default
{
          "id": "full-screen-widget-default",
          "created": "2025-11-08T23:32:10.584881+08:00",
          "updated": "2025-11-08T23:32:50.772022+08:00",
         "_system": {
                              "owner_id": "$[[SETUP_OWNER_ID]]"
                            },
          "payload": {
            "ai_overview": {
              "enabled": false,
              "height": 200,
              "logo": {},
              "title": "AI Overview"
            },
            "ai_widgets": {
              "enabled": false,
              "widgets": []
            },
            "logo": {}
          },
          "type": "page",
          "name": "Coco Search",
          "enabled_module": {
            "search": {
              "enabled": true,
              "datasource": [
                "*"
              ],
              "placeholder": "Search whatever you want..."
            },
            "ai_chat": {
              "enabled": false,
              "start_page_config": {
                "enabled": false,
                "logo": {
                  "light": "",
                  "dark": ""
                },
                "introduction": "",
                "display_assistants": null
              }
            }
          },
          "access_control": {
            "authentication": false,
            "chat_history": false
          },
          "appearance": {
            "theme": "auto",
            "language": "zh-CN"
          },
          "cors": {
            "enabled": true,
            "allowed_origins": [
              "*"
            ]
          },
          "guest": {
            "enabled": false,
            "run_as": "5f67d03147dfce10ed51feafd346c8ce"
          },
          "enabled": true
        }