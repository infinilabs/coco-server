POST $[[SETUP_INDEX_PREFIX]]app-roles/$[[SETUP_DOC_TYPE]]/d47m4df3edbo74oibebg
{
          "id": "d47m4df3edbo74oibebg",
          "created": "2025-11-08T23:31:01.927656+08:00",
          "updated": "2025-11-08T23:31:01.927656+08:00",
           "_system": {
                      "owner_id": "$[[SETUP_OWNER_ID]]"
                    },
          "name": "Guest",
          "description": "",
          "grants": {
            "permissions": [
               "coco#connector/read",
                "coco#datasource/read",
                "coco#document/read",
                "coco#integration/read",
                "coco#mcp_server/read",
                "coco#model_provider/read",
                "coco#system/read",
                "generic#entity:card/read",
                "generic#entity:label/read",
                "generic#security:authorization/read",
                "generic#security:permission/read",
                "generic#security:principal/search",
                "generic#security:role/read",
                "generic#security:user/read",
                "generic#sharing/read",
                "coco#assistant/read",
                "coco#assistant/search",
                "coco#session/create",
                "coco#session/read",
                "coco#session/search",
                "coco#session/view_single_session_history",
                "coco#assistant/ask",
                "coco#search/search",
                "coco#datasource/search",
                "coco#connector/search",
                "coco#document/search",
                "coco#mcp_server/search"
            ]
          }
}

POST $[[SETUP_INDEX_PREFIX]]app-roles/$[[SETUP_DOC_TYPE]]/d47m83v3edbo74oibfc0
{
          "id": "d47m83v3edbo74oibfc0",
          "created": "2025-11-08T23:38:55.688787+08:00",
          "updated": "2025-11-08T23:38:55.688787+08:00",
      "_system": {
                     "owner_id": "$[[SETUP_OWNER_ID]]"
                   },
          "name": "admin",
          "description": "System administrator role with full permissions",
          "grants": {
            "permissions": [
                "*.*"
            ]
          }
}