import { Tabs } from "antd";
import styles from "./index.module.less";

export function Categories(props) {

  const { category = "all", onChange } = props;

  return (
    <Tabs className={styles.categories} activeKey={category || "all"} items={[
      {
        key: 'all',
        label: '全部',
      },
      {
        key: 'doc',
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
