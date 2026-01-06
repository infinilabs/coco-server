import { Tabs } from "antd";
import styles from "./index.module.less";

export function Categories(props) {

  const { type = "all", onChange } = props;

  return (
    <Tabs className={styles.categories} activeKey={type} items={[
      {
        key: 'all',
        label: '全部',
      },
      {
        key: 'file',
        label: '文档',
      },
      {
        key: 'image',
        label: '图片',
      },
      {
        key: 'video',
        label: '视频',
      },
    ]} onChange={onChange} />
  )
}

export default Categories;
