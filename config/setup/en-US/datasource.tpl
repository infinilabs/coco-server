POST $[[SETUP_INDEX_PREFIX]]datasource$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/coco_server_docs
{
  "id" : "coco_server_docs",
  "created" : "2025-03-04T14:26:23.43811+08:00",
  "updated" : "2025-03-04T14:26:23.439214+08:00",
  "type" : "connector",
  "name" : "Coco Server Docs",
  "icon" : "font_coco",
  "connector" : {
    "id" : "hugo_site",
    "config" : {
      "interval" : "600m",
      "sync_type" : "interval",
      "urls" : [
        "https://docs.infinilabs.com/coco-server/main/index.json"
      ]
    }
  },
  "sync_enabled" : true,
  "enabled" : true
}

POST $[[SETUP_INDEX_PREFIX]]datasource$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/coco_app_docs
{
  "id" : "coco_app_docs",
  "created" : "2025-03-04T14:25:23.43811+08:00",
  "updated" : "2025-03-04T14:25:23.439214+08:00",
  "type" : "connector",
  "name" : "Coco App Docs",
  "icon" : "https://coco.rs/favicon.ico",
  "connector" : {
    "id" : "hugo_site",
    "config" : {
      "interval" : "600m",
      "sync_type" : "interval",
      "urls" : [
        "https://docs.infinilabs.com/coco-app/main/index.json"
      ]
    }
  },
  "sync_enabled" : true,
  "enabled" : true
}