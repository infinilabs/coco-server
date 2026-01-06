import { List, Spin, Drawer } from "antd";
import { memo, useMemo, useState } from "react";
import { X } from "lucide-react";
import ResultDetail from "../ResultDetail";
import styles from "./NormalList.module.less";
import SearchResults from "@infinilabs/search-results";
import { DocDetail } from "@infinilabs/doc-detail";

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
    let currentType = data[0].type;
    let tempGroup = [data[0]];

    for (let i = 1; i < data.length; i++) {
      const currentItem = data[i];
      const isCurrentAllowed = ALLOWED_MERGE_TYPES.includes(currentItem.type);
      const isLastAllowed = ALLOWED_MERGE_TYPES.includes(currentType);

      if (isCurrentAllowed && currentItem.type === currentType) {
        tempGroup.push(currentItem);
      } else {
        if (isLastAllowed && tempGroup.length > 1) {
          result.push(tempGroup);
        } else {
          result.push(...tempGroup);
        }
        currentType = currentItem.type;
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
                      onClick: () => { }
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
              <ResultDetail
                key={item.id}
                getContainer={getDetailContainer}
                data={item}
                isMobile={isMobile}
              >
                <SearchResults
                  section={item}
                  onRecordClick={(record) => {
                    onOpen(record)
                  }}
                />
              </ResultDetail>
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
      <Drawer
        onClose={onClose}
        open={open}
        width={isMobile ? '100%' : 800}
        closeIcon={null}
        rootClassName={styles.detail}
        getContainer={getDetailContainer}
        destroyOnHidden
        classNames={{
          wrapper: `!overflow-hidden ${isMobile ? '!left-12px !right-12px !w-[calc(100%-24px)]' : '!right-24px'} !top-146px !bottom-24px !rounded-12px !shadow-[0_2px_20px_rgba(0,0,0,0.1)] !dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)]`,
          body: '!p-24px !rounded-12px'
        }}
        mask={false}
        maskClosable={false}
      >
        <X className="color-[#bbb] cursor-pointer absolute right-24px top-24px z-1" onClick={onClose} />
        <DocDetail data={record || {}}/>
      </Drawer>
    </>
  );
}

export default memo(NormalList);