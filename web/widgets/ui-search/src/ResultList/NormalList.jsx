import { List, Typography, Tag } from "antd";
import { memo } from "react";
import { SquareArrowOutUpRight } from "lucide-react";

import ResultDetail from "../ResultDetail";
import { formatDate, isWithin7Days } from "../utils/date";
import { highlightText } from "./highlight";

import styles from "./NormalList.module.less";

export function NormalList(props) {
  const { getDetailContainer, from, size, hits, onSearch, isMobile, query } = props;

  const searchQuery = query || '';

  return (
    <div className={styles.list}>
      <List
        itemLayout="vertical"
        size="large"
        pagination={{
          onChange: (page, pageSize) => {
            onSearch((page - 1) * pageSize, pageSize);
          },
          pageSize: size,
          current: Math.round(from / size + 1),
          total: hits?.total || 0,
        }}
        dataSource={hits?.hits || []}
        renderItem={(item, index) => (
          <ResultDetail
            key={item.id}
            getContainer={getDetailContainer}
            data={item}
            isMobile={isMobile}
          >
            <List.Item
              actions={[]}
              className="mb-24px"
              extra={
                item.thumbnail ? (
                  <img width="100%" src={item.thumbnail} />
                ) : null
              }
            >
              <List.Item.Meta
                title={
                  <a
                    title={item.title}
                    className="flex items-center text-base text-[#333]"
                  >
                    {item.icon?.startsWith("http") && (
                      <img src={item.icon} className="mr-1 w-4 h-4" />
                    )}
                    {highlightText(item.title, searchQuery)}
                  </a>
                }
                description={
                  item.summary ? (
                    <Typography.Text
                      ellipsis={{
                        suffix: "...",
                        ellipsis: true,
                        rows: 3,
                        expanded: true,
                      }}
                      className="text-[#999]"
                    >
                      {highlightText(item.summary, searchQuery)}
                    </Typography.Text>
                  ) : null
                }
              />
              <div className="flex items-center mb-6px">
                {item.source?.name && (
                  <Tag color="#58a65c" className="text-xs">
                    {highlightText(item.source.name, searchQuery)}
                  </Tag>
                )}
                <Tag bordered={false} color="blue" className="text-[##101010]">
                  {highlightText(item.type, searchQuery)}
                </Tag>
                {item.last_updated_by?.user?.username &&
                item.last_updated_by?.timestamp ? (
                  <span className="text-[##101010]">
                    Last updated by {highlightText(item.last_updated_by.user.username, searchQuery)}{" "}
                    {isWithin7Days(item.last_updated_by?.timestamp)
                      ? formatDate(item.last_updated_by?.timestamp)
                      : `at ${item.last_updated_by?.timestamp}`}
                  </span>
                ) : (
                  item.created && (
                    <span className="text-[##101010]">
                      Created{" "}
                      {isWithin7Days(item.created)
                        ? formatDate(item.created)
                        : `at ${item.created}`}
                    </span>
                  )
                )}
              </div>
              {item.url && (
                <div className="mb-6px">
                  {item.url && (
                    <a
                      className="truncate w-full inline-block align-middle"
                      href={item.url}
                      target="_blank"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <SquareArrowOutUpRight className="relative top-2px w-14px h-14px mr-1" />
                      {highlightText(item.url, searchQuery)}
                    </a>
                  )}
                </div>
              )}
            </List.Item>
          </ResultDetail>
        )}
      />
    </div>
  );
}

export default memo(NormalList);
