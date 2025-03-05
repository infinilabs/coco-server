import hugoSiteBlog from "@/assets/connector/hugo_site/blog.png"; 
import hugoSiteIcon from "@/assets/connector/hugo_site/icon.png"; 
import hugoSiteNews from "@/assets/connector/hugo_site/news.png"; 
import hugoSiteWebpage from "@/assets/connector/hugo_site/web_page.png";
import hugoSiteWeb from "@/assets/connector/hugo_site/web.png"; 

import gdAudio from "@/assets/connector/google_drive/audio.png"; 
import gdDocment from "@/assets/connector/google_drive/document.png"; 
import gdDrawing from "@/assets/connector/google_drive/drawing.png"; 
import gdFolder from "@/assets/connector/google_drive/folder.png";
import gdForm from "@/assets/connector/google_drive/form.png"; 
import gdFusiontable from "@/assets/connector/google_drive/fusiontable.png";
import gdIcon from "@/assets/connector/google_drive/icon.png"; 
import gdJam from "@/assets/connector/google_drive/jam.png";
import gdMap from "@/assets/connector/google_drive/map.png"; 
import gdMSExcel from "@/assets/connector/google_drive/ms_excel.png";
import gdMSPowerpoint from "@/assets/connector/google_drive/ms_powerpoint.png"; 
import gdMSWord from "@/assets/connector/google_drive/ms_word.png";
import gdPDF from "@/assets/connector/google_drive/pdf.png"; 
import gdPhoto from "@/assets/connector/google_drive/photo.png";
import gdPresentation from "@/assets/connector/google_drive/presentation.png"; 
import gdScript from "@/assets/connector/google_drive/script.png";
import gdSite from "@/assets/connector/google_drive/site.png"; 
import gdSpreadsheet from "@/assets/connector/google_drive/spreadsheet.png";
import gdVideo from "@/assets/connector/google_drive/video.png";
import gdZip from "@/assets/connector/google_drive/zip.png";

import notionDatabase from "@/assets/connector/notion/database.png";
import notionIcon from "@/assets/connector/notion/icon.png";
import notionPage from "@/assets/connector/notion/page.png";

import yuqueBoard from "@/assets/connector/yuque/board.png";
import yuqueBook from "@/assets/connector/yuque/book.png";
import yuqueDirectory from "@/assets/connector/yuque/directory.png";
import yuqueDoc from "@/assets/connector/yuque/doc.png";
import yuqueFolder from "@/assets/connector/yuque/folder.png";
import yuqueIcon from "@/assets/connector/yuque/icon.png";
import yuqueSheet from "@/assets/connector/yuque/sheet.png";
import yuqueTable from "@/assets/connector/yuque/table.png";

const ConnectorIcons = {
  "google_drive": {
    "audio": gdAudio,
    "document": gdDocment,
    "drawing": gdDrawing,
    "folder": gdFolder,
    "form": gdForm,
    "fusiontable": gdFusiontable,
    "icon": gdIcon,
    "jam": gdJam,
    "map": gdMap,
    "ms_excel": gdMSExcel,
    "ms_powerpoint": gdMSPowerpoint,
    "ms_word": gdMSWord,
    "pdf": gdPDF,
    "photo": gdPhoto,
    "presentation": gdPresentation,
    "script": gdScript,
    "site": gdSite,
    "spreadsheet": gdSpreadsheet,
    "video": gdVideo,
    "zip": gdZip,
  },
  "hugo_site": {
    "blog": hugoSiteBlog,
    "icon": hugoSiteIcon,
    "news": hugoSiteNews,
    "web_page": hugoSiteWebpage,
    "web": hugoSiteWeb,
  },
  "notion": {
    "database": notionDatabase,
    "icon": notionIcon,
    "page": notionPage,
  },
  "yuque": {
    "board": yuqueBoard,
    "book": yuqueBook,
    "directory": yuqueDirectory,
    "doc": yuqueDoc,
    "folder": yuqueFolder,
    "icon": yuqueIcon,
    "sheet": yuqueSheet,
    "table": yuqueTable,
  }
}



export const ConnectorImageIcon = ({connector, doc_type }) => {
  const iconImage = ConnectorIcons[connector]?.[doc_type];
  if(!iconImage) {
    return null;
  }
  return <img src={iconImage} alt="icon" style={{ width: "1em", height: "1em" }} />
};
