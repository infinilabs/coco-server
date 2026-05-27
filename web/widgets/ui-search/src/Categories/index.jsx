import { Tabs } from "antd";
import styles from "./index.module.less";
import { useTranslation } from 'react-i18next';

export function Categories(props) {

  const { category = "all", onChange } = props;

  const { t } = useTranslation();

  return (
    <Tabs className={styles.categories} activeKey={category || "all"} items={[
      {
        key: 'all',
        label: t('labels.all'),
      },
      {
        key: 'doc',
        label: t('labels.document'),
      },
      {
        key: 'image',
        label: t('labels.image'),
      },
      // {
      //   key: 'video',
      //   label: '视频',
      // },
    ]} onChange={onChange} />
  )
}

export default Categories;
