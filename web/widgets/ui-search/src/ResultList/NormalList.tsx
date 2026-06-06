import { List, Spin } from "antd";
import { memo, useEffect, useState } from "react";
import ResultDetail from "../ResultDetail";
import styles from "./NormalList.module.less";
import SearchResults from "@infinilabs/search-results";

interface NormalListProps {
  getDetailContainer?: () => HTMLElement;
  data?: Record<string, any>[];
  isMobile?: boolean;
  query?: string;
  loading?: boolean;
  hasMore?: boolean;
  setDetailCollapse?: (v: boolean) => void;
  getRawContent?: (data: Record<string, any>) => string;
  apiConfig?: Record<string, any>;
  [key: string]: any;
}

export function NormalList(props: NormalListProps) {
  const {
    getDetailContainer,
    data = [],
    isMobile,
    query,
    loading,
    hasMore,
    setDetailCollapse,
    getRawContent,
    apiConfig
  } = props;

  const [open, setOpen] = useState(false);
  const [record, setRecord] = useState<Record<string, any> | undefined>();

  useEffect(() => {
    setOpen(false);
    setRecord(undefined);
    setDetailCollapse?.(false);
  }, [data]);

  const onOpen = (record: Record<string, any>) => {
    setRecord(record);
    setOpen(true);
    setDetailCollapse?.(true)
  };

  const onClose = () => {
    setOpen(false);
    setRecord(undefined);
    setDetailCollapse?.(false)
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
                } as any}
                onRecordClick={(record: any) => {
                  onOpen(record)
                }}
                requestHeaders={apiConfig?.headers}
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
        apiConfig={apiConfig}
      />
    </>
  );
}

export default memo(NormalList);