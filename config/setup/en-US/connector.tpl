POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/yuque
{
  "id" : "yuque",
  "created" : "2025-03-04T14:26:23.43811+08:00",
  "updated" : "2025-03-04T14:26:23.439214+08:00",
  "name" : "Yuque Docs Connector",
  "description" : "Fetch the docs metadata for yuque.",
  "category" : "website",
  "icon" : "/assets/icons/connector/yuque/icon.png",
  "tags" : [
    "static_site",
    "hugo",
    "web"
  ],
  "url" : "http://coco.rs/connectors/hugo_site",
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
  "builtin": true
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/hugo_site
{
  "id" : "hugo_site",
  "created" : "2025-03-04T14:27:41.869073+08:00",
  "updated" : "2025-03-04T14:27:41.869288+08:00",
  "name" : "Hugo Site Connector",
  "description" : "Fetch the index.json file from a specified Hugo site.",
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
  "builtin": true
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/google_drive
{
  "id" : "google_drive",
  "created" : "2025-03-04T15:27:11.359656+08:00",
  "updated" : "2025-03-04T15:27:11.359753+08:00",
  "name" : "Google Drive Connector",
  "description" : "Fetch the files metadata from Google Drive.",
  "category" : "cloud_storage",
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
  "builtin": true
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/notion
{
  "id" : "notion",
  "created" : "2025-03-04T15:27:26.620836+08:00",
  "updated" : "2025-03-04T15:27:26.620918+08:00",
  "name" : "Notion Docs Connector",
  "description" : "Fetch the docs metadata for notion.",
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
  "builtin": true
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/rss
{
  "id" : "rss",
  "created" : "2025-07-14T20:50:00.869073+08:00",
  "updated" : "2025-07-14T20:50:00.869073+08:00",
  "name" : "RSS Connector",
  "description" : "Fetch items from a specified RSS feed.",
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
  "builtin": true
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/local_fs
{
  "id" : "local_fs",
  "created" : "2025-07-18T10:00:00.000000+08:00",
  "updated" : "2025-07-18T10:00:00.000000+08:00",
  "name" : "Local Filesystem Connector",
  "description" : "Scan and fetch metadata from local files.",
  "category" : "local_storage",
  "icon" : "/assets/icons/connector/local_fs/icon.png",
  "tags" : [
    "storage",
    "filesystem"
  ],
  "url" : "http://coco.rs/connectors/local_fs",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/local_fs/icon.png"
    }
  },
  "builtin": true
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/s3
{
  "id" : "s3",
  "created" : "2025-07-24T00:00:00.000000+08:00",
  "updated" : "2025-07-24T00:00:00.000000+08:00",
  "name" : "S3 Storage Connector",
  "description" : "Fetch S3 Storage objects metadata.",
  "category" : "cloud_storage",
  "icon" : "/assets/icons/connector/s3/icon.png",
  "tags" : [
    "s3",
    "storage"
  ],
  "url" : "http://coco.rs/connectors/s3",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/s3/icon.png"
    }
  },
  "builtin": true
}
POST $[[SETUP_INDEX_PREFIX]]connector$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/confluence
{
  "id" : "confluence",
  "created" : "2025-07-30T00:00:00.000000+08:00",
  "updated" : "2025-07-30T00:00:00.000000+08:00",
  "name" : "Confluence wiki Connector",
  "description" : "Fetch Confluence wiki data.",
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
  "builtin": true
}