POST $[[SETUP_INDEX_PREFIX]]datasource$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/coco_server_docs
{
  "id" : "coco_server_docs",
  "created" : "2025-03-04T14:26:23.43811+08:00",
  "updated" : "2025-03-04T14:26:23.439214+08:00",
  "type" : "connector",
  "name" : "Coco Server 文档",
  "icon" : "font_coco",
  "connector" : {
    "id" : "hugo_site",
    "config" : {
      "urls" : [
        "https://docs.infinilabs.com/coco-server/main/index.json"
      ]
    }
  },
  "sync" : {
    "enabled": true,
    "interval" : "600m",
    "strategy" : "interval"
  },
  "enabled" : true
}

POST $[[SETUP_INDEX_PREFIX]]datasource$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/coco_app_docs
{
  "id" : "coco_app_docs",
  "created" : "2025-03-04T14:25:23.43811+08:00",
  "updated" : "2025-03-04T14:25:23.439214+08:00",
  "type" : "connector",
  "name" : "Coco App 文档",
  "icon" : "https://coco.rs/favicon.ico",
  "connector" : {
    "id" : "hugo_site",
    "config" : {
      "urls" : [
        "https://docs.infinilabs.com/coco-app/main/index.json"
      ]
    }
  },
  "sync" : {
    "enabled": true,
    "interval" : "600m",
    "strategy" : "interval"
  },
  "enabled" : true
}

POST $[[SETUP_INDEX_PREFIX]]datasource$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/hacker_news
{
  "id" : "hacker_news",
  "created" : "2025-03-04T14:25:23.43811+08:00",
  "updated" : "2025-03-04T14:25:23.439214+08:00",
  "type" : "connector",
  "name" : "Hacker News",
  "icon" : "https://news.ycombinator.com/favicon.ico",
  "connector" : {
    "id" : "rss",
    "config" : {
      "urls" : [
        "https://news.ycombinator.com/rss"
      ]
    }
  },
  "sync" : {
    "enabled": true,
    "interval" : "600m",
    "strategy" : "interval"
  },
  "enabled" : true
}