import { List, Spin } from "antd";
import { memo, useState } from "react";
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
    setDetailCollapse,
    getRawContent
  } = props;

  const [open, setOpen] = useState(false);
  const [record, setRecord] = useState();

  const onOpen = (record) => {
    setRecord(record);
    setOpen(true);
    setDetailCollapse(true)
  };

  const onClose = () => {
    setOpen(false);
    setRecord();
    setDetailCollapse(false)
  };

  return (
    <>
      <div className={styles.list}>
        <List
          itemLayout="vertical"
          size="large"
          pagination={false}
          dataSource={data || []}
          renderItem={(item, index) => {
            const isActive = item.id === record?.id
            return (
              <SearchResults
                section={{
                  ...item,
                  isActive
                }}
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
        getRawContent={getRawContent}
      />
    </>
  );
}

export default memo(NormalList);