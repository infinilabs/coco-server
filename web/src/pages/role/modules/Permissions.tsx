import { Checkbox, Collapse, Spin, Table, Tabs } from "antd";
import styles from "./Permissions.module.less";
import { fetchPermissions } from "@/service/api/role";

export default (props) => {
  const { value = {}, onChange, filters, teamID } = props;
  const { feature: filterFeatures } = filters || {};
  const { feature = [] } = value;
  const [loading, setLoading] = useState(false);
  const [features, setFeatures] = useState([]);
  const { t } = useTranslation();

  const handleCheckAll = (checked, operations, feature) => {
    const newFeature = [...feature];
    if (checked) {
      operations.forEach((o) => {
        const index = newFeature.indexOf(o.id);
        if (index === -1) {
          newFeature.push(o.id);
        }
      });
    } else {
      operations.forEach((o) => {
        const index = newFeature.indexOf(o.id);
        if (index !== -1) {
          newFeature.splice(index, 1);
        }
      });
    }
    onChange({
      ...value,
      feature: newFeature,
    });
  };

  const handleCheck = (checked, item, feature) => {
    const newFeature = [...feature];
    const index = newFeature.indexOf(item.id);
    if (checked) {
      if (index === -1) {
        newFeature.push(item.id);
      }
    } else {
      if (index !== -1) {
        newFeature.splice(index, 1);
      }
    }
    onChange({
      ...value,
      feature: newFeature,
    });
  };

  const getFeatures = async () => {
    try {
      setLoading(true);
      
      // const res = await fetchPermissions();
      const res = [
  {
    "id": "coco:home/view",
    "description": "",
    "category": "coco:home",
    "order": "001001001"
  },
  {
    "id": "coco:ai_assistant/view",
    "description": "",
    "category": "coco:ai_assistant",
    "order": "001002001"
  },
  {
    "id": "coco:ai_assistant/create",
    "description": "",
    "category": "coco:ai_assistant",
    "order": "001002002"
  },
  {
    "id": "coco:ai_assistant/update",
    "description": "",
    "category": "coco:ai_assistant",
    "order": "001002003"
  },
  {
    "id": "coco:ai_assistant/delete",
    "description": "",
    "category": "coco:ai_assistant",
    "order": "001002004"
  },
  {
    "id": "coco:mcp/view",
    "description": "",
    "category": "coco:mcp",
    "order": "001003001"
  },
  {
    "id": "coco:mcp/create",
    "description": "",
    "category": "coco:mcp",
    "order": "001003002"
  },
  {
    "id": "coco:mcp/update",
    "description": "",
    "category": "coco:mcp",
    "order": "001003003"
  },
  {
    "id": "coco:mcp/delete",
    "description": "",
    "category": "coco:mcp",
    "order": "001003004"
  },
  {
    "id": "coco:datasource/view",
    "description": "",
    "category": "coco:datasource",
    "order": "001004001"
  },
  {
    "id": "coco:datasource/create",
    "description": "",
    "category": "coco:datasource",
    "order": "001004002"
  },
  {
    "id": "coco:datasource/update",
    "description": "",
    "category": "coco:datasource",
    "order": "001004003"
  },
  {
    "id": "coco:datasource/delete",
    "description": "",
    "category": "coco:datasource",
    "order": "001004004"
  },
  {
    "id": "coco:api_token/view",
    "description": "",
    "category": "coco:api_token",
    "order": "001005001"
  },
  {
    "id": "coco:api_token/create",
    "description": "",
    "category": "coco:api_token",
    "order": "001005002"
  },
  {
    "id": "coco:api_token/update",
    "description": "",
    "category": "coco:api_token",
    "order": "001005003"
  },
  {
    "id": "coco:api_token/delete",
    "description": "",
    "category": "coco:api_token",
    "order": "001005004"
  },
  {
    "id": "coco:integration/view",
    "description": "",
    "category": "coco:integration",
    "order": "001006001"
  },
  {
    "id": "coco:integration/create",
    "description": "",
    "category": "coco:integration",
    "order": "001006002"
  },
  {
    "id": "coco:integration/update",
    "description": "",
    "category": "coco:integration",
    "order": "001006003"
  },
  {
    "id": "coco:integration/delete",
    "description": "",
    "category": "coco:integration",
    "order": "001006004"
  },
  {
    "id": "coco:role/view",
    "description": "",
    "category": "coco:security",
    "order": "001007001"
  },
  {
    "id": "coco:role/create",
    "description": "",
    "category": "coco:security",
    "order": "001007002"
  },
  {
    "id": "coco:role/update",
    "description": "",
    "category": "coco:security",
    "order": "001007003"
  },
  {
    "id": "coco:role/delete",
    "description": "",
    "category": "coco:security",
    "order": "001007004"
  },
  {
    "id": "coco:user/view",
    "description": "",
    "category": "coco:security",
    "order": "001008001"
  },
  {
    "id": "coco:user/create",
    "description": "",
    "category": "coco:security",
    "order": "001008002"
  },
  {
    "id": "coco:user/update",
    "description": "",
    "category": "coco:security",
    "order": "001008003"
  },
  {
    "id": "coco:user/delete",
    "description": "",
    "category": "coco:security",
    "order": "001008004"
  },
  {
    "id": "coco:connector/view",
    "description": "",
    "category": "coco:settings",
    "order": "001009001"
  },
  {
    "id": "coco:connector/create",
    "description": "",
    "category": "coco:settings",
    "order": "001009002"
  },
  {
    "id": "coco:connector/update",
    "description": "",
    "category": "coco:settings",
    "order": "001009003"
  },
  {
    "id": "coco:connector/delete",
    "description": "",
    "category": "coco:settings",
    "order": "001009004"
  },
  {
    "id": "coco:server_settings/view",
    "description": "",
    "category": "coco:settings",
    "order": "001010001"
  },
  {
    "id": "coco:server_settings/update",
    "description": "",
    "category": "coco:settings",
    "order": "001010002"
  },
  {
    "id": "coco:app_settings/view",
    "description": "",
    "category": "coco:settings",
    "order": "001011001"
  },
  {
    "id": "coco:app_settings/update",
    "description": "",
    "category": "coco:settings",
    "order": "001011002"
  },
  {
    "id": "coco:search_settings/view",
    "description": "",
    "category": "coco:settings",
    "order": "001012001"
  },
  {
    "id": "coco:search_settings/update",
    "description": "",
    "category": "coco:settings",
    "order": "001012002"
  },
]
      if (Array.isArray(res)) {
        const sortedFeatures = res.sort((a, b) => a.order - b.order);
        setFeatures(sortedFeatures);
  
        const filteredFeatures = feature.filter((item) => 
          res.some((r) => r.id === item)
        );
  
        onChange({
          ...value,
          feature: filteredFeatures,
        });
      } else {
        setFeatures([]);
      }
    } catch (error) {
      setFeatures([]);
    } finally {
      setLoading(false);
    }
  };
  

  const buildTreeData = (features, filters, teamID) => {
    const treeData = [];
    features
      .filter((item) => {
        if (!teamID) return !!item.category;
        return !!item.category && (filters || []).indexOf(item.id) !== -1;
      })
      .forEach((item) => {
        const categories = item.category.split(":");
        if (categories.length === 2) {
          const root = categories[0];
          const subRoot = categories[1];
          const index = treeData.findIndex((t) => t.key === root);
          if (index === -1) {
            treeData.push({
              key: root,
              data: [
                {
                  object: subRoot,
                  operations: [item],
                },
              ],
            });
          } else {
            const subIndex = treeData[index].data.findIndex(
              (t) => t.object === subRoot
            );
            if (subIndex === -1) {
              treeData[index].data.push({
                object: subRoot,
                operations: [item],
              });
            } else {
              treeData[index].data[subIndex].operations.push(item);
            }
          }
        }
      });
    return treeData;
  };

  const renderCount = (item, feature) => {
    let total = 0;
    let checked = 0;
    const operations = [];
    item.data?.forEach((d) => {
      d.operations?.forEach((o) => {
        total++;
        operations.push(o);
        if (feature.indexOf(o.id) !== -1) {
          checked++;
        }
      });
      d.children?.forEach((c) => {
        c.operations?.forEach((o) => {
          total++;
          operations.push(o);
          if (feature.indexOf(o.id) !== -1) {
            checked++;
          }
        });
      })
    });
    return (
      <>
        <span className={styles.count}>{`${checked}/${total}`}</span>
        <Checkbox
          indeterminate={checked > 0 && checked !== total}
          checked={checked === total}
          onChange={(e) =>
            handleCheckAll(e.target.checked, operations, feature)
          }
          onClick={(e) => {
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
  }, [JSON.stringify(features), JSON.stringify(filterFeatures), teamID]);

  if (loading) {
    return <Spin spinning={true} />;
  }
  
  const items = useMemo(() => {
    return treeData.map((item) => {
      const featureComponent = (
        <Table
          rowKey={"object"}
          size="small"
          columns={[
            {
              key: "object",
              dataIndex: "object",
              title: t('page.role.labels.object'),
              width: 100,
              render: (text, record) => {
                return t(`permission.${text}`);
              },
            },
            {
              key: "operations",
              dataIndex: "operations",
              title: t('common.operation'),
              className: styles.operations,
              render: (text, record) => {
                return text?.map((item) => {
                  let label = t(`permission.${item.id}`);
                  return (
                    <Checkbox
                      key={item.id}
                      checked={feature.indexOf(item.id) !== -1}
                      onChange={(e) =>
                        handleCheck(e.target.checked, item, feature)
                      }
                    >
                      {label}
                    </Checkbox>
                  );
                });
              },
            },
            {
              key: "checkAll",
              dataIndex: "checkAll",
              title: "",
              width: 16,
              render: (text, record) => {
                let total = 0;
                let checked = 0;
                let operations = record.operations || []
                operations.forEach((o) => {
                  total++;
                  if (feature.indexOf(o.id) !== -1) {
                    checked++;
                  }
                });
                return (
                  <Checkbox
                    indeterminate={checked > 0 && checked !== total}
                    checked={checked === total}
                    onChange={(e) =>
                      handleCheckAll(e.target.checked, operations, feature)
                    }
                  ></Checkbox>
                );
              },
            },
          ]}
          dataSource={item.data}
          pagination={false}
        />
      );
      return {
        key: item.key,
        label: (
          <div className={styles.header}>
            {t(`page.role.labels.${item.key}`)}
          </div>
        ),
        extra: renderCount(item, feature),
        children: featureComponent,
      }
    })
  }, [treeData])

  return (
    <Collapse size="small" items={items} className={styles.permissions} accordion />
  );
};
