import { Checkbox, Collapse, Spin, Table } from 'antd';
import type { Key } from 'react';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { fetchPermissions } from '@/service/api/role';

import styles from './Permissions.module.less';

interface PermissionsProps {
  readonly value?: { feature?: string[] };
  readonly onChange?: (value: { feature?: string[] }) => void;
  readonly filters?: { feature?: string[] };
  readonly teamID?: string;
}

type FeatureItem = {
  id: string;
  category: string;
  resource: string;
  action: string;
};

export default function Permissions(props: PermissionsProps) {
  const { value = {}, onChange, filters, teamID } = props;
  const { feature: filterFeatures } = filters || {};
  const { feature = [] } = value;
  const [loading, setLoading] = useState(false);
  const [features, setFeatures] = useState<FeatureItem[]>([]);
  const { t } = useTranslation();

  // eslint-disable-next-line @typescript-eslint/no-shadow
  const handleCheckAll = (checked: boolean, operations: any[], feature: string[]) => {
    const newFeature = [...feature];
    if (checked) {
      operations.forEach((o: { id: any }) => {
        const index = newFeature.indexOf(o.id);
        if (index === -1) {
          newFeature.push(o.id);
        }
      });
    } else {
      operations.forEach((o: { id: any }) => {
        const index = newFeature.indexOf(o.id);
        if (index !== -1) {
          newFeature.splice(index, 1);
        }
      });
    }
    if (onChange) {
      onChange({
        ...value,
        feature: newFeature
      });
    }
  };

  // eslint-disable-next-line @typescript-eslint/no-shadow
  const handleCheck = (checked: boolean, item: { id: any }, feature: string[]) => {
    const newFeature = [...feature];
    const index = newFeature.indexOf(item.id);
    if (checked) {
      if (index === -1) {
        newFeature.push(item.id);
      }
    } else if (index !== -1) {
      newFeature.splice(index, 1);
    }
    if (onChange) {
      onChange({
        ...value,
        feature: newFeature
      });
    }
  };

  const getFeatures = async () => {
    try {
      setLoading(true);

      const res: any = await fetchPermissions();
      if (Array.isArray(res.data)) {
        setFeatures(res.data);

        // eslint-disable-next-line prettier/prettier
        const filteredFeatures = feature.filter(item =>
          res.data.some((r: { id: string }) => r.id === item)
        );

        if (onChange) {
          onChange({
            ...value,
            feature: filteredFeatures
          });
        }
      } else {
        setFeatures([]);
      }
    } catch (error) {
      console.error(error);
      setFeatures([]);
    } finally {
      setLoading(false);
    }
  };

  // eslint-disable-next-line @typescript-eslint/no-shadow
  const buildTreeData = (features: FeatureItem[], filters?: string[], teamID?: string) => {
    const treeData: Array<{ key: string; data: Array<{ object: string; operations: FeatureItem[] }> }> = [];
    features
      .filter(item => {
        if (!item.category) return false;
        // teamID 存在时，仅当 filters 提供且非空才按 filters 过滤
        if (teamID) {
          if (filters && filters.length > 0) {
            return (filters || []).includes(item.id);
          }
          return true;
        }
        // 无 teamID 时显示所有有 category 的条目
        return true;
      })
      .forEach(item => {
        // 新格式按 category → resource 分组
        const root = item.category;
        const subRoot = item.resource;
        // eslint-disable-next-line @typescript-eslint/no-shadow
        const index = treeData.findIndex(t => t.key === root);
        if (index === -1) {
          treeData.push({
            key: root,
            data: [
              {
                object: subRoot,
                operations: [item]
              }
            ]
          });
        } else {
          // eslint-disable-next-line @typescript-eslint/no-shadow
          const subIndex = treeData[index].data.findIndex(t => t.object === subRoot);
          if (subIndex === -1) {
            treeData[index].data.push({
              object: subRoot,
              operations: [item]
            });
          } else {
            treeData[index].data[subIndex].operations.push(item);
          }
        }
      });
    return treeData;
  };

  // eslint-disable-next-line @typescript-eslint/no-shadow
  const renderCount = (item: { key?: string; data: any }, feature: string | any[]) => {
    let total = 0;
    let checked = 0;
    const operations: any[] = [];
    item.data?.forEach((d: { operations: any[]; children: any[] }) => {
      d.operations?.forEach((o: { id: any }) => {
        // eslint-disable-next-line no-plusplus
        total++;
        operations.push(o);
        if (feature.includes(o.id)) {
          // eslint-disable-next-line no-plusplus
          checked++;
        }
      });
      d.children?.forEach((c: { operations: any[] }) => {
        c.operations?.forEach((o: { id: any }) => {
          // eslint-disable-next-line no-plusplus
          total++;
          operations.push(o);
          if (feature.includes(o.id)) {
            // eslint-disable-next-line no-plusplus
            checked++;
          }
        });
      });
    });
    return (
      <>
        <span className={styles.count}>{`${checked}/${total}`}</span>
        <Checkbox
          checked={checked === total}
          indeterminate={checked > 0 && checked !== total}
          onChange={e => handleCheckAll(e.target.checked, operations, feature as string[])}
          onClick={e => {
            e.stopPropagation();
          }}
        />
      </>
    );
  };

  useEffect(() => {
    getFeatures();
  }, []);

  const treeData = useMemo(() => {
    return buildTreeData(features, filterFeatures, teamID);
  }, [features, filterFeatures, teamID]);

  const items = useMemo(() => {
    return treeData.map(item => {
      const featureComponent = (
        <Table
          dataSource={item.data}
          pagination={false}
          rowKey='object'
          size='small'
          columns={[
            {
              key: 'object',
              dataIndex: 'object',
              title: t('page.role.labels.object'),
              width: 100,
              render: text => {
                // 显示 resource 的文案
                return t(`permission.${text}`);
              }
            },
            {
              key: 'operations',
              dataIndex: 'operations',
              title: t('common.operation'),
              className: styles.operations,
              render: ops => {
                // 每个 operation 是一个 { id, category, resource, action }
                return ops?.map((op: { id: Key | null | undefined }) => {
                  const label = t(`permission.${op.id}`);
                  return (
                    <Checkbox
                      checked={feature.includes(op.id as string)}
                      key={op.id}
                      onChange={e => handleCheck(e.target.checked, op, feature)}
                    >
                      {label}
                    </Checkbox>
                  );
                });
              }
            },
            {
              key: 'checkAll',
              dataIndex: 'checkAll',
              title: '',
              width: 16,
              render: (_text, record) => {
                let total = 0;
                let checked = 0;
                const operations = record.operations || [];
                operations.forEach(o => {
                  // eslint-disable-next-line no-plusplus
                  total++;
                  if (feature.includes(o.id)) {
                    // eslint-disable-next-line no-plusplus
                    checked++;
                  }
                });
                return (
                  <Checkbox
                    checked={checked === total}
                    indeterminate={checked > 0 && checked !== total}
                    onChange={e => handleCheckAll(e.target.checked, operations, feature)}
                  />
                );
              }
            }
          ]}
        />
      );
      return {
        key: item.key,
        label: <div className={styles.header}>{t(`page.role.labels.${item.key}`)}</div>,
        extra: renderCount(item, feature),
        children: featureComponent
      };
    });
  }, [treeData, feature, t]);

  // 早退放在所有 Hook 之后
  if (loading) {
    return <Spin spinning={true} />;
  }

  return (
    <Collapse
      accordion
      className={styles.permissions}
      items={items}
      size='small'
    />
  );
}
