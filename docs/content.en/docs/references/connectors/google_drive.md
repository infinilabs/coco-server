---
title: "Google Drive"
weight: 10
---

# Google Drive Connector

## Register Google Drive Connector

```shell
curl -XPUT http://localhost:9000/connector/google_drive?replace=true -d '{
    "name": "Google Drive Connector", 
    "description": "Fetch the files metadata from Google Drive.", 
    "icon": "https://coco.infini.cloud/assets/connector/google_drive/icon.png", 
    "category": "cloud_storage", 
    "tags": [
        "google", 
        "storage"
    ], 
    "url": "http://coco.rs/connectors/google_drive", 
    "assets": {
        "icons": {
            "default": "https://coco.infini.cloud/assets/connector/google_drive/icon.png", 
            "audio": "https://coco.infini.cloud/assets/connector/google_drive/audio.png", 
            "form": "https://coco.infini.cloud/assets/connector/google_drive/form.png", 
            "document": "https://coco.infini.cloud/assets/connector/google_drive/document.png", 
            "drawing": "https://coco.infini.cloud/assets/connector/google_drive/drawing.png", 
            "folder": "https://coco.infini.cloud/assets/connector/google_drive/folder.png", 
             "fusiontable": "https://coco.infini.cloud/assets/connector/google_drive/fusiontable.png", 
             "jam": "https://coco.infini.cloud/assets/connector/google_drive/jam.png", 
             "map": "https://coco.infini.cloud/assets/connector/google_drive/map.png", 
             "ms_excel": "https://coco.infini.cloud/assets/connector/google_drive/ms_excel.png", 
             "ms_powerpoint": "https://coco.infini.cloud/assets/connector/google_drive/ms_powerpoint.png", 
             "ms_word": "https://coco.infini.cloud/assets/connector/google_drive/ms_word.png", 
             "pdf": "https://coco.infini.cloud/assets/connector/google_drive/pdf.png", 
             "photo": "https://coco.infini.cloud/assets/connector/google_drive/photo.png", 
            "presentation": "https://coco.infini.cloud/assets/connector/google_drive/presentation.png", 
            "script": "https://coco.infini.cloud/assets/connector/google_drive/script.png", 
            "site": "https://coco.infini.cloud/assets/connector/google_drive/site.png", 
            "spreadsheet": "https://coco.infini.cloud/assets/connector/google_drive/spreadsheet.png",
            "video": "https://coco.infini.cloud/assets/connector/google_drive/video.png",
            "zip": "https://coco.infini.cloud/assets/connector/google_drive/zip.png"
        }
    }
}'
```

> Use `google_drive` as a unique identifier, or substitute it with any ID of your choice.
>
> Replace `https://coco.infini.cloud` to your coco-server's endpoint.
