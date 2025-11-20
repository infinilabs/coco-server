POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/yuque
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "yuque",
  "created" : "2025-03-04T14:26:23.43811+08:00",
  "updated" : "2025-03-04T14:26:23.439214+08:00",
  "name" : "语雀文档连接器",
  "description" : "从语雀中提取文档的元数据信息。",
  "category" : "website",
  "icon" : "/assets/icons/connector/yuque/icon.png",
  "tags" : [
    "static_site",
    "hugo",
    "web"
  ],
  "url" : "http://coco.rs/connectors/yuque",
  "assets" : {
    "icons" : {
      "board" : "/assets/icons/connector/yuque/board.png",
      "book" : "/assets/icons/connector/yuque/book.png",
      "default" : "/assets/icons/connector/yuque/icon.png",
      "doc" : "/assets/icons/connector/yuque/doc.png",
      "sheet" : "/assets/icons/connector/yuque/sheet.png",
      "table" : "/assets/icons/connector/yuque/table.png"
    }
  },
  "builtin": true,
   "processor":{
       "enabled":true,
       "name":"yuque"
    }
}

POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/hugo_site
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "hugo_site",
  "created" : "2025-03-04T14:27:41.869073+08:00",
  "updated" : "2025-03-04T14:27:41.869288+08:00",
  "name" : "Hugo 站点连接器",
  "description" : "提取指定 Hugo 站点的 index.json 文件。",
  "category" : "website",
  "icon" : "/assets/icons/connector/hugo_site/icon.png",
  "tags" : [
    "static_site",
    "hugo",
    "web"
  ],
  "url" : "http://coco.rs/connectors/hugo_site",
  "assets" : {
    "icons" : {
      "blog" : "/assets/icons/connector/hugo_site/blog.png",
      "default" : "/assets/icons/connector/hugo_site/web.png",
      "news" : "/assets/icons/connector/hugo_site/news.png",
      "web" : "/assets/icons/connector/hugo_site/web.png",
      "web_page" : "/assets/icons/connector/hugo_site/web_page.png"
    }
  },
  "builtin": true,
  "processor":{
      "enabled":true,
      "name":"hugo_site"
  }
}

POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/google_drive
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "google_drive",
  "created" : "2025-03-04T15:27:11.359656+08:00",
  "updated" : "2025-03-04T15:27:11.359753+08:00",
  "name" : "Google 云盘连接器",
  "description" : "提取 Google 云端硬盘中文件的元数据。",
  "category" : "cloud_storage",
  "path_hierarchy":false,
  "icon" : "/assets/icons/connector/google_drive/icon.png",
  "tags" : [
    "google",
    "storage"
  ],
  "url" : "http://coco.rs/connectors/google_drive",
  "assets" : {
    "icons" : {
      "audio" : "/assets/icons/connector/google_drive/audio.png",
      "default" : "/assets/icons/connector/google_drive/icon.png",
      "document" : "/assets/icons/connector/google_drive/document.png",
      "drawing" : "/assets/icons/connector/google_drive/drawing.png",
      "folder" : "/assets/icons/connector/google_drive/folder.png",
      "form" : "/assets/icons/connector/google_drive/form.png",
      "fusiontable" : "/assets/icons/connector/google_drive/fusiontable.png",
      "jam" : "/assets/icons/connector/google_drive/jam.png",
      "map" : "/assets/icons/connector/google_drive/map.png",
      "ms_excel" : "/assets/icons/connector/google_drive/ms_excel.png",
      "ms_powerpoint" : "/assets/icons/connector/google_drive/ms_powerpoint.png",
      "ms_word" : "/assets/icons/connector/google_drive/ms_word.png",
      "pdf" : "/assets/icons/connector/google_drive/pdf.png",
      "photo" : "/assets/icons/connector/google_drive/photo.png",
      "presentation" : "/assets/icons/connector/google_drive/presentation.png",
      "script" : "/assets/icons/connector/google_drive/script.png",
      "site" : "/assets/icons/connector/google_drive/site.png",
      "spreadsheet" : "/assets/icons/connector/google_drive/spreadsheet.png",
      "video" : "/assets/icons/connector/google_drive/video.png",
      "zip" : "/assets/icons/connector/google_drive/zip.png"
    }
  },
  "config": {
    "auth_url": "https://accounts.google.com/o/oauth2/auth",
    "redirect_url": "$[[SETUP_SERVER_ENDPOINT]]/connector/google_drive/oauth_redirect",
    "token_url": "https://oauth2.googleapis.com/token"
  },
  "builtin": true,
  "oauth_connect_implemented": true,
  "processor":{
      "enabled":true,
      "name":"google_drive"
  }
}

POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/notion
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "notion",
  "created" : "2025-03-04T15:27:26.620836+08:00",
  "updated" : "2025-03-04T15:27:26.620918+08:00",
  "name" : "Notion 文档连接器",
  "description" : "提取 Notion 文档元数据信息。",
  "category" : "website",
  "icon" : "/assets/icons/connector/notion/icon.png",
  "tags" : [
    "docs",
    "notion",
    "web"
  ],
  "url" : "http://coco.rs/connectors/notion",
  "assets" : {
    "icons" : {
      "database" : "/assets/icons/connector/notion/database.png",
      "default" : "/assets/icons/connector/notion/icon.png",
      "page" : "/assets/icons/connector/notion/page.png",
      "web_page" : "/assets/icons/connector/notion/icon.png"
    }
  },
  "builtin": true,
    "processor":{
       "enabled":true,
       "name":"notion"
    }
}

POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/rss
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "rss",
  "created" : "2025-07-14T20:50:00.869073+08:00",
  "updated" : "2025-07-14T20:50:00.869073+08:00",
  "name" : "RSS 连接器",
  "description" : "提取 RSS feed 文档信息。",
  "category" : "website",
  "icon" : "/assets/icons/connector/rss/icon.png",
  "tags" : [
    "rss",
    "feed",
    "web"
  ],
  "url" : "http://coco.rs/connectors/rss",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/rss/icon.png"
    }
  },
   "builtin": true,
     "processor":{
        "enabled":true,
        "name":"rss"
     }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/local_fs
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "local_fs",
  "created" : "2025-07-18T10:00:00.000000+08:00",
  "updated" : "2025-07-18T10:00:00.000000+08:00",
  "name" : "本地文件连接器",
  "description" : "提取本地文件元数据信息。",
  "category" : "local_storage",
  "path_hierarchy":false,
  "icon" : "/assets/icons/connector/local_fs/icon.png",
  "tags" : [
    "storage",
    "filesystem"
  ],
  "url" : "http://coco.rs/connectors/local_fs",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/local_fs/icon.png",
      "file" : "/assets/icons/connector/local_fs/file.png"
    }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "local_fs"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/s3
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "s3",
  "created" : "2025-07-24T00:00:00.000000+08:00",
  "updated" : "2025-07-24T00:00:00.000000+08:00",
  "name" : "S3 对象存储连接器",
  "description" : "提取 S3 云端文件元数据信息。",
  "category" : "cloud_storage",
  "path_hierarchy":false,
  "icon" : "/assets/icons/connector/s3/icon.png",
  "tags" : [
    "s3",
    "storage"
  ],
  "url" : "http://coco.rs/connectors/s3",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/s3/icon.png",
      "file" : "/assets/icons/connector/s3/file.png"
    }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "s3"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/confluence
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "confluence",
  "created" : "2025-07-30T00:00:00.000000+08:00",
  "updated" : "2025-07-30T00:00:00.000000+08:00",
  "name" : "Confluence wiki 连接器",
  "description" : "提取 Confluence wiki 数据信息。",
  "category" : "website",
  "icon" : "/assets/icons/connector/confluence/icon.png",
  "tags" : [
    "wiki",
    "storage",
    "docs",
    "web"
  ],
  "url" : "http://coco.rs/connectors/confluence",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/confluence/icon.png"
    }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "confluence"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/network_drive
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "network_drive",
  "created" : "2025-08-05T00:00:00.000000+08:00",
  "updated" : "2025-08-05T00:00:00.000000+08:00",
  "name" : "网络驱动器",
  "description" : "扫描网络共享文件并提取元数据。",
  "category" : "cloud_storage",
  "path_hierarchy":false,
  "icon" : "/assets/icons/connector/network_drive/icon.png",
  "tags" : [
    "filesystem",
    "storage",
    "web"
  ],
  "url" : "http://coco.rs/connectors/network_drive",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/network_drive/icon.png",
      "file" : "/assets/icons/connector/network_drive/file.png"
    }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "network_drive"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/postgresql
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "postgresql",
  "created" : "2025-08-14T00:00:00.000000+08:00",
  "updated" : "2025-08-14T00:00:00.000000+08:00",
  "name" : "PostgreSQL 连接器",
  "description" : "提取 PostgreSQL 数据库数据。",
  "category" : "database",
  "icon" : "/assets/icons/connector/postgresql/icon.png",
  "tags" : [
    "sql",
    "storage",
    "database"
  ],
  "url" : "http://coco.rs/connectors/postgresql",
  "assets" : {
     "icons" : {
       "default" : "/assets/icons/connector/postgresql/icon.png"
     }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "postgresql"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/mysql
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "mysql",
  "created" : "2025-08-22T00:00:00.000000+08:00",
  "updated" : "2025-08-22T00:00:00.000000+08:00",
  "name" : "MySQL 连接器",
  "description" : "提取 MySQL 数据库数据。",
  "category" : "database",
  "icon" : "/assets/icons/connector/mysql/icon.png",
  "tags" : [
    "sql",
    "storage",
    "database"
  ],
  "url" : "http://coco.rs/connectors/mysql",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/mysql/icon.png"
    }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "mysql"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/github
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "github",
  "created" : "2025-08-25T00:00:00.000000+08:00",
  "updated" : "2025-08-25T00:00:00.000000+08:00",
  "name" : "GitHub 连接器",
  "description" : "提取 GitHub repositories, issues 和 pull requests。",
  "category" : "website",
  "path_hierarchy":false,
  "icon" : "/assets/icons/connector/github/icon.png",
  "tags" : [
    "git",
    "code",
    "vcs",
    "website"
  ],
  "url" : "http://coco.rs/connectors/github",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/github/icon.png",
      "repository" : "/assets/icons/connector/github/repository.png",
      "issue" : "/assets/icons/connector/github/issue.png",
      "pull_request" : "/assets/icons/connector/github/pull_request.png",
      "org" : "/assets/icons/connector/github/org.png"
    }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "github"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/gitlab
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "gitlab",
  "created" : "2025-08-29T00:00:00.000000+08:00",
  "updated" : "2025-08-29T00:00:00.000000+08:00",
  "name" : "GitLab 连接器",
  "description" : "提取 GitLab repositories, issues, 和 merge requests。",
  "category" : "website",
  "path_hierarchy":false,
  "icon" : "/assets/icons/connector/gitlab/icon.png",
  "tags" : [
    "git",
    "code",
    "vcs",
    "website"
  ],
  "url" : "http://coco.rs/connectors/gitlab",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/gitlab/icon.png",
      "repository" : "/assets/icons/connector/gitlab/repository.png",
      "issue" : "/assets/icons/connector/gitlab/issue.png",
      "merge_request" : "/assets/icons/connector/gitlab/merge_request.png",
      "wiki" : "/assets/icons/connector/gitlab/wiki.png",
      "snippet" : "/assets/icons/connector/gitlab/snippet.png",
      "org" : "/assets/icons/connector/gitlab/org.png"
    }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "gitlab"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/gitea
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "gitea",
  "created" : "2025-09-04T00:00:00.000000+08:00",
  "updated" : "2025-09-04T00:00:00.000000+08:00",
  "name" : "Gitea 连接器",
  "description" : "提取 Gitea repositories, issues, 和 pull requests。",
  "category" : "website",
  "path_hierarchy":false,
  "icon" : "/assets/icons/connector/gitea/icon.png",
  "tags" : [
    "git",
    "code",
    "vcs",
    "website"
  ],
  "url" : "http://coco.rs/connectors/gitea",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/gitea/icon.png",
      "repository" : "/assets/icons/connector/gitea/repository.png",
      "issue" : "/assets/icons/connector/gitea/issue.png",
      "pull_request" : "/assets/icons/connector/gitea/pull_request.png",
      "org" : "/assets/icons/connector/gitea/org.png"
    }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "gitea"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/feishu
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id": "feishu",
  "created": "2025-08-22T00:00:00.000000+08:00",
  "updated": "2025-08-22T00:00:00.000000+08:00",
  "name": "飞书云文档连接器",
  "description": "索引飞书云文档，包括文档、表格、思维笔记、多维表格和知识库等。",
  "category": "cloud",
  "icon": "/assets/icons/connector/feishu/icon.png",
  "tags": [
    "feishu",
    "cloud_docs"
  ],
  "url": "http://coco.rs/connectors/feishu",
  "assets": {
    "icons": {
      "default": "/assets/icons/connector/feishu/icon.png",
      "doc": "/assets/icons/connector/feishu/doc.png",
      "docx":  "/assets/icons/connector/feishu/docx.png",
      "sheet": "/assets/icons/connector/feishu/sheet.png",
      "mindnote": "/assets/icons/connector/feishu/mindnote.png",
      "bitable": "/assets/icons/connector/feishu/bitable.png",
      "file": "/assets/icons/connector/feishu/file.png",
      "slides": "/assets/icons/connector/feishu/slides.png"
    }
  },
  "config": {
    "redirect_uri": "$[[SETUP_SERVER_ENDPOINT]]/connector/feishu/oauth_redirect",
    "auth_url": "https://accounts.feishu.cn/open-apis/authen/v1/authorize",
    "token_url": "https://open.feishu.cn/open-apis/authen/v2/oauth/token"
  },
  "builtin": true,
  "path_hierarchy":false,
  "oauth_connect_implemented": true,
  "processor": {
    "enabled": true,
    "name": "feishu"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/lark
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id": "lark",
  "created": "2025-09-01T00:00:00.000000+08:00",
  "updated": "2025-09-01T00:00:00.000000+08:00",
  "name": "Lark 云文档连接器",
  "description": "索引 Lark 云文档，包括文档、表格、思维笔记、多维表格和知识库等。",
  "category": "cloud",
  "icon": "/assets/icons/connector/lark/icon.png",
  "tags": [
    "lark",
    "cloud_docs"
  ],
  "url": "http://coco.rs/connectors/lark",
  "assets": {
    "icons": {
      "default": "/assets/icons/connector/lark/icon.png",
      "doc": "/assets/icons/connector/lark/doc.png",
      "docx":  "/assets/icons/connector/lark/docx.png",
      "sheet": "/assets/icons/connector/lark/sheet.png",
      "mindnote": "/assets/icons/connector/lark/mindnote.png",
      "bitable": "/assets/icons/connector/lark/bitable.png",
      "file": "/assets/icons/connector/lark/file.png",
      "slides": "/assets/icons/connector/lark/slides.png"
    }
  },
  "config": {
    "redirect_uri": "$[[SETUP_SERVER_ENDPOINT]]/connector/lark/oauth_redirect",
    "auth_url": "https://accounts.larksuite.com/open-apis/authen/v1/authorize",
    "token_url": "https://open.larksuite.com/open-apis/authen/v2/oauth/token"
  },
  "builtin": true,
  "path_hierarchy":false,
  "oauth_connect_implemented": true,
  "processor": {
    "enabled": true,
    "name": "lark"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/mssql
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "mssql",
  "created" : "2025-09-15T00:00:00.000000+08:00",
  "updated" : "2025-09-15T00:00:00.000000+08:00",
  "name" : "Microsoft SQL Server 连接器",
  "description" : "提取 Microsoft SQL Server 数据库数据。",
  "category" : "database",
  "icon" : "/assets/icons/connector/mssql/icon.png",
  "tags" : [
    "sql",
    "storage",
    "database"
  ],
  "url" : "http://coco.rs/connectors/mssql",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/mssql/icon.png"
    }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "mssql"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/oracle
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "oracle",
  "created" : "2025-09-19T00:00:00.000000+08:00",
  "updated" : "2025-09-19T00:00:00.000000+08:00",
  "name" : "Oracle 连接器",
  "description" : "提取 Oracle 数据库数据。",
  "category" : "database",
  "icon" : "/assets/icons/connector/oracle/icon.png",
  "tags" : [
    "sql",
    "storage",
    "database"
  ],
  "url" : "http://coco.rs/connectors/oracle",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/oracle/icon.png"
    }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "oracle"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/salesforce
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "salesforce",
  "created" : "2025-09-08T00:00:00.000000+08:00",
  "updated" : "2025-09-08T00:00:00.000000+08:00",
  "name" : "Salesforce 连接器",
  "description" : "与 Salesforce 集成，通过智能字段缓存和查询优化从您的 Salesforce 组织中索引和搜索数据。",
  "category" : "crm",
  "path_hierarchy" : true,
  "icon" : "/assets/icons/connector/salesforce/icon.png",
  "tags" : [
    "crm",
    "salesforce",
    "business"
  ],
  "url" : "http://coco.rs/connectors/salesforce",
  "assets" : {
    "icons" : {
      "account" : "/assets/icons/connector/salesforce/account.png",
      "campaign" : "/assets/icons/connector/salesforce/campaign.png",
      "case" : "/assets/icons/connector/salesforce/case.png",
      "contact" : "/assets/icons/connector/salesforce/contact.png",
      "default" : "/assets/icons/connector/salesforce/icon.png",
      "lead" : "/assets/icons/connector/salesforce/lead.png",
      "opportunity" : "/assets/icons/connector/salesforce/opportunity.png"
    }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "salesforce"
  }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/neo4j
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "neo4j",
  "created" : "2025-09-29T00:00:00.000000+08:00",
  "updated" : "2025-09-29T00:00:00.000000+08:00",
  "name" : "Neo4j 图数据库连接器",
  "description" : "连接到 Neo4j 图数据库，使用自定义 Cypher 查询索引节点，支持增量同步和分页功能。",
  "category" : "database",
  "icon" : "/assets/icons/connector/neo4j/icon.png",
  "tags" : [
    "graph",
    "database",
    "neo4j",
    "cypher"
  ],
  "url" : "http://coco.rs/connectors/neo4j",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/neo4j/icon.png"
    }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "neo4j"
  }
}

POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/mongodb
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "mongodb",
  "created" : "2025-10-26T00:00:00.000000+08:00",
  "updated" : "2025-10-26T00:00:00.000000+08:00",
  "name" : "MongoDB 连接器",
  "description" : "从 MongoDB 集合中提取文档，支持基于时间戳或 ObjectId 的增量同步。",
  "category" : "database",
  "icon" : "/assets/icons/connector/mongodb/icon.png",
  "tags" : [
    "nosql",
    "database",
    "mongodb",
    "document_store"
  ],
  "url" : "http://coco.rs/connectors/mongodb",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/mongodb/icon.png"
    }
  },
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "mongodb"
  }
}

POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/gitlab_webhook_receiver
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "gitlab_webhook_receiver",
    "name" : "Gitlab Webhook（Merge Request）连接器",
    "description" : "通过 Webhook 来接收来自 Gitlab Merge Requests 来进行代码的自动 Review。",
  "created" : "2025-09-29T00:00:00.000000+08:00",
  "updated" : "2025-09-29T00:00:00.000000+08:00",
    "builtin": true,
    "icon": "font_gitlab",
    "oauth_connect_implemented": false,
    "path_hierarchy": false,
    "assets": {
      "icons": {
        "default": "/assets/icons/connector/gitlab/icon.png"
      }
    },
    "category": "Webhook",
    "processor": {
        "enabled": false
      }
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/box
{
  "id" : "box",
  "created" : "2025-11-02T00:00:00.000000+08:00",
  "updated" : "2025-11-02T00:00:00.000000+08:00",
  "name" : "Box 云存储连接器",
  "description" : "索引 Box 中的文件和文件夹，支持免费账号和企业账号的多用户访问。",
  "category" : "cloud_storage",
  "path_hierarchy" : true,
  "icon" : "/assets/icons/connector/box/icon.png",
  "tags" : [
    "box",
    "cloud_storage",
    "file_sharing"
  ],
  "url" : "http://coco.rs/connectors/box",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/box/icon.png",
      "bookmark" : "/assets/icons/connector/box/bookmark.png",
      "boxcanvas" : "/assets/icons/connector/box/boxcanvas.png",
      "boxnote" : "/assets/icons/connector/box/boxnote.png",
      "docx" : "/assets/icons/connector/box/docx.png",
      "excel-spreadsheet" : "/assets/icons/connector/box/excel-spreadsheet.png",
      "google-docs" : "/assets/icons/connector/box/google-docs.png",
      "google-sheets" : "/assets/icons/connector/box/google-sheets.png",
      "google-slides" : "/assets/icons/connector/box/google-slides.png",
      "keynote" : "/assets/icons/connector/box/keynote.png",
      "numbers" : "/assets/icons/connector/box/numbers.png",
      "pages" : "/assets/icons/connector/box/pages.png",
      "pdf" : "/assets/icons/connector/box/pdf.png",
      "powerpoint-presentation" : "/assets/icons/connector/box/powerpoint-presentation.png"
    }
  },
  "config": {
    "redirect_uri": "$[[SETUP_SERVER_ENDPOINT]]/connector/box/oauth_redirect"
  },
  "builtin": true,
  "oauth_connect_implemented": true,
  "processor": {
    "enabled": true,
    "name": "box"
  }
}