import { Select, Image } from "antd"
// import { ReactSVG } from 'react-svg';
// import PersonalDriveSVG from '@/assets/svg-icon/connector/personal-drive.svg';
// import ChengGuoTuiGuangSVG from '@/assets/svg-icon/connector/cgtgpx.svg';
// import FeishuSVG from '@/assets/svg-icon/connector/feishu.svg';
// import GithubSVG from '@/assets/svg-icon/connector/github.svg';
// import JinshanDocsSVG from '@/assets/svg-icon/connector/jinshan-docs.svg';
// import ObsidianSVG from '@/assets/svg-icon/connector/obsidian.svg';
// import OnenoteSVG from '@/assets/svg-icon/connector/onenote.svg';
// import QingJingZhiShiSVG from '@/assets/svg-icon/connector/qjzstj.svg';
// import ShiMoSVG from '@/assets/svg-icon/connector/shimo.svg';
// import SimpleNoteSVG from '@/assets/svg-icon/connector/simple-note.svg';
// import TencentDocsSVG from '@/assets/svg-icon/connector/tencent-docs.svg';
// import YingXiangNoteSVG from '@/assets/svg-icon/connector/yingxiang-note.svg';
// import YoudaoSVG from '@/assets/svg-icon/connector/youdao.svg';
// import ZhiNengWenDaSVG from '@/assets/svg-icon/connector/znwd.svg';

// import CompressedSVG from '@/assets/svg-icon/file/compressed.svg';
// import ExcelSVG from '@/assets/svg-icon/file/excel.svg';
// import ImageSVG from '@/assets/svg-icon/file/image.svg';
// import FolderSVG from '@/assets/svg-icon/file/folder.svg';
// import LinkSVG from '@/assets/svg-icon/file/link.svg';
// import PdfSVG from '@/assets/svg-icon/file/pdf.svg';
// import PowerpointSVG from '@/assets/svg-icon/file/powerpoint.svg';
// import TextSVG from '@/assets/svg-icon/file/text.svg';
// import VideoSVG from '@/assets/svg-icon/file/video.svg';
// import WordSVG from '@/assets/svg-icon/file/word.svg';
// import SoundSVG from '@/assets/svg-icon/file/sound.svg';
// import UnkonwSVG from '@/assets/svg-icon/file/unknown.svg';

// const IconMap = {
//   connector: {
//     'personal-drive': PersonalDriveSVG,
//     'cgtgpx': ChengGuoTuiGuangSVG,
//     'feishu': FeishuSVG,
//     'github': GithubSVG,
//     'jinshan-docs': JinshanDocsSVG,
//     'obsidian': ObsidianSVG,
//     'onenote': OnenoteSVG,
//     'qjzstj': QingJingZhiShiSVG,
//     'shimo': ShiMoSVG,
//     'simple-note': SimpleNoteSVG,
//     'tencent-docs': TencentDocsSVG,
//     'yingxiang-note': YingXiangNoteSVG,
//     'youdao': YoudaoSVG,
//     'znwd': ZhiNengWenDaSVG,
//   },
//   file: {
//     'compressed': CompressedSVG,
//     'excel': ExcelSVG,
//     'image': ImageSVG,
//     'folder': FolderSVG,
//     'link': LinkSVG,
//     'pdf': PdfSVG,
//     'powerpoint': PowerpointSVG,
//     'text': TextSVG,
//     'video': VideoSVG,
//     'word': WordSVG,
//     'sound': SoundSVG,
//     'unknown': UnkonwSVG,
//   }
// }

export const IconSelector = ({value, onChange, className, icons=[]})=> {
  return <Select showSearch={true} value={value}  className={className} onChange={onChange}>
    {icons.map(icon => {
        return <Select.Option value={icon.path} item={icon} >
          <div className="flex items-center gap-3px"><Image preview={false} width="1em" height="1em" src={icon.path}/><span>{icon.name}</span></div>
        </Select.Option>
    })}
  </Select>
}