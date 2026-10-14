import { Tabs } from "antd";
import styles from "./index.module.less";
import { useTranslation } from 'react-i18next';

interface CategoriesProps {
  category?: string;
  /** tab keys worth showing — derived from what the result set actually
   * contains, so users never switch to a tab that is empty (unset = all) */
  categories?: string[];
  onChange?: (key: string) => void;
}

export function Categories(props: CategoriesProps) {

  const { category = "all", categories, onChange } = props;

  const { t } = useTranslation();

  const allItems = [
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
  ];

  return (
    <Tabs
      className={styles.categories}
      activeKey={category || "all"}
      items={!categories || categories.length === 0 ? allItems : allItems.filter(item => categories.includes(item.key))}
      onChange={onChange}
    />
  )
}

export default Categories;
