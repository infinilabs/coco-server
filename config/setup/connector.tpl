POST $[[SETUP_INDEX_PREFIX]]connector/$[[SETUP_DOC_TYPE]]/yuque
{
  "id" : "yuque",
  "created" : "2025-03-04T14:26:23.43811+08:00",
  "updated" : "2025-03-04T14:26:23.439214+08:00",
  "name" : "Yuque Docs Connector",
  "description" : "Fetch the docs metadata for yuque.",
  "category" : "website",
  "icon" : "/assets/connector/yuque/icon.png",
  "tags" : [
    "static_site",
    "hugo",
    "web"
  ],
  "url" : "http://coco.rs/connectors/hugo_site",
  "assets" : {
    "icons" : {
      "board" : "/assets/connector/yuque/board.png",
      "book" : "/assets/connector/yuque/book.png",
      "default" : "/assets/connector/yuque/icon.png",
      "doc" : "/assets/connector/yuque/doc.png",
      "sheet" : "/assets/connector/yuque/sheet.png",
      "table" : "/assets/connector/yuque/table.png"
    }
  },
  "builtin": true
}
POST $[[SETUP_INDEX_PREFIX]]connector/$[[SETUP_DOC_TYPE]]/hugo_site
{
  "id" : "hugo_site",
  "created" : "2025-03-04T14:27:41.869073+08:00",
  "updated" : "2025-03-04T14:27:41.869288+08:00",
  "name" : "Hugo Site Connector",
  "description" : "Fetch the index.json file from a specified Hugo site.",
  "category" : "website",
  "icon" : "/assets/connector/hugo_site/icon.png",
  "tags" : [
    "static_site",
    "hugo",
    "web"
  ],
  "url" : "http://coco.rs/connectors/hugo_site",
  "assets" : {
    "icons" : {
      "blog" : "/assets/connector/hugo_site/blog.png",
      "default" : "/assets/connector/hugo_site/web.png",
      "news" : "/assets/connector/hugo_site/news.png",
      "web" : "/assets/connector/hugo_site/web.png",
      "web_page" : "/assets/connector/hugo_site/web_page.png"
    }
  },
  "builtin": true
}
POST $[[SETUP_INDEX_PREFIX]]connector/$[[SETUP_DOC_TYPE]]/google_drive
{
  "id" : "google_drive",
  "created" : "2025-03-04T15:27:11.359656+08:00",
  "updated" : "2025-03-04T15:27:11.359753+08:00",
  "name" : "Google Drive Connector",
  "description" : "Fetch the files metadata from Google Drive.",
  "category" : "cloud_storage",
  "icon" : "/assets/connector/google_drive/icon.png",
  "tags" : [
    "google",
    "storage"
  ],
  "url" : "http://coco.rs/connectors/google_drive",
  "assets" : {
    "icons" : {
      "audio" : "/assets/connector/google_drive/audio.png",
      "default" : "/assets/connector/google_drive/icon.png",
      "document" : "/assets/connector/google_drive/document.png",
      "drawing" : "/assets/connector/google_drive/drawing.png",
      "folder" : "/assets/connector/google_drive/folder.png",
      "form" : "/assets/connector/google_drive/form.png",
      "fusiontable" : "/assets/connector/google_drive/fusiontable.png",
      "jam" : "/assets/connector/google_drive/jam.png",
      "map" : "/assets/connector/google_drive/map.png",
      "ms_excel" : "/assets/connector/google_drive/ms_excel.png",
      "ms_powerpoint" : "/assets/connector/google_drive/ms_powerpoint.png",
      "ms_word" : "/assets/connector/google_drive/ms_word.png",
      "pdf" : "/assets/connector/google_drive/pdf.png",
      "photo" : "/assets/connector/google_drive/photo.png",
      "presentation" : "/assets/connector/google_drive/presentation.png",
      "script" : "/assets/connector/google_drive/script.png",
      "site" : "/assets/connector/google_drive/site.png",
      "spreadsheet" : "/assets/connector/google_drive/spreadsheet.png",
      "video" : "/assets/connector/google_drive/video.png",
      "zip" : "/assets/connector/google_drive/zip.png"
    }
  },
  "builtin": true
}
POST $[[SETUP_INDEX_PREFIX]]connector/$[[SETUP_DOC_TYPE]]/notion
{
  "id" : "notion",
  "created" : "2025-03-04T15:27:26.620836+08:00",
  "updated" : "2025-03-04T15:27:26.620918+08:00",
  "name" : "Notion Docs Connector",
  "description" : "Fetch the docs metadata for notion.",
  "category" : "website",
  "icon" : "/assets/connector/notion/icon.png",
  "tags" : [
    "docs",
    "notion",
    "web"
  ],
  "url" : "http://coco.rs/connectors/notion",
  "assets" : {
    "icons" : {
      "database" : "/assets/connector/notion/database.png",
      "default" : "/assets/connector/notion/icon.png",
      "page" : "/assets/connector/notion/page.png",
      "web_page" : "/assets/connector/notion/icon.png"
    }
  },
  "builtin": true
}