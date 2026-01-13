import { List, Spin } from "antd";
import { memo, useMemo, useState } from "react";
import ResultDetail from "../ResultDetail";
import styles from "./NormalList.module.less";
import SearchResults from "@infinilabs/search-results";

export function NormalList(props) {
  const {
    getDetailContainer,
    data = [],
    isMobile,
    query,
    loading,
    hasMore,
  } = props;

  const [open, setOpen] = useState(false);
  const [record, setRecord] = useState();

  const onOpen = (record) => {
    setRecord(record);
    setOpen(true);
  };

  const onClose = () => {
    setOpen(false);
    setRecord();
  };

  const formatData = useMemo(() => {
    if (!Array.isArray(data) || data.length === 0) return [];
    const ALLOWED_MERGE_TYPES = ['image', 'video'];
    const result = [];
    let currentType = data[0].metadata?.content_type;
    let tempGroup = [data[0]];

    for (let i = 1; i < data.length; i++) {
      const currentItem = data[i];
      const isCurrentAllowed = ALLOWED_MERGE_TYPES.includes(currentItem.metadata?.content_type);
      const isLastAllowed = ALLOWED_MERGE_TYPES.includes(currentType);

      if (isCurrentAllowed && currentItem.metadata?.content_type === currentType) {
        tempGroup.push(currentItem);
      } else {
        if (isLastAllowed && tempGroup.length > 1) {
          result.push(tempGroup);
        } else {
          result.push(...tempGroup);
        }
        currentType = currentItem.metadata?.content_type;
        tempGroup = [currentItem];
      }
    }

    const isLastGroupAllowed = ALLOWED_MERGE_TYPES.includes(currentType);
    if (isLastGroupAllowed && tempGroup.length > 1) {
      result.push(tempGroup);
    } else {
      result.push(...tempGroup);
    }
    
    return result;
  }, [data])

  return (
    <>
      <div className={styles.list}>
        <List
          itemLayout="vertical"
          size="large"
          pagination={false}
          dataSource={formatData || []}
          renderItem={(item, index) => {
            if (Array.isArray(item)) {
              return (
                <div >
                  <SearchResults
                    footerAction={{
                      label: "More",
                      onClick: () => { console.log("more...") }
                    }}
                    section={item}
                    onRecordClick={(record) => {
                      if (typeof record.url === "string") window.open(record.url);
                    }}
                  />
                </div>
              )
            }
            return (
              <SearchResults
                section={item}
                onRecordClick={(record) => {
                  onOpen(record)
                }}
              />
            )
          }}
        />
        {loading && hasMore && (
          <div style={{
            textAlign: 'center',
            padding: '16px 0',
            marginTop: '8px',
          }}>
            <Spin />
          </div>
        )}
      </div>
      <ResultDetail 
        getContainer={getDetailContainer}
        open={open}
        onClose={onClose}
        data={record || {}}
        isMobile={isMobile}
      />
    </>
  );
}

export default memo(NormalList);