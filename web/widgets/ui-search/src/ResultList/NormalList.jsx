import { List } from "antd";
import styles from "./NormalList.module.less"
import ResultDetail from "../ResultDetail";
import { memo } from "react";
import { Tag } from "antd";
import { formatDate, isWithin7Days } from "../utils/date";
import { SquareArrowOutUpRight } from "lucide-react";
import { Typography } from "antd";

export function NormalList(props) {

    const { getDetailContainer, from, size, hits, onSearch, isMobile } = props

    return (
        <div className={styles.list}>
            <List
                itemLayout="vertical"
                size="large"
                pagination={{
                    onChange: (page, pageSize) => {
                        onSearch((page - 1) * pageSize, pageSize)
                    },
                    pageSize: size,
                    current: Math.round(from / size + 1),
                    total: hits?.total || 0
                }}
                dataSource={hits?.hits || []}
                renderItem={(item, index) => (
                    <ResultDetail key={item.id} getContainer={getDetailContainer} data={item} isMobile={isMobile}>
                         <List.Item
                            actions={[]}
                            className="mb-24px"
                            extra={
                                item.thumbnail ? (
                                    <img
                                        width="100%"
                                        src={item.thumbnail}
                                    />
                                ) : null
                            }
                        >
                            <List.Item.Meta
                                title={(
                                    <a title={item.title}>
                                        { item.icon?.startsWith('http') && <img src={item.icon} className="mr-4px w-20px h-20px"/> }
                                        {item.title}
                                    </a>
                                )}
                                description={item.summary ? <Typography.Text ellipsis={{ suffix: '...', ellipsis: true, rows: 3, expanded: true }}>{item.summary}</Typography.Text> : null}
                            />
                            <div className="mb-6px">
                                { item.source?.name && <Tag color="#58a65c">{item.source?.name}</Tag> }
                                { item.last_updated_by?.user?.username && item.last_updated_by?.timestamp ? (
                                    <span>Last updated by {item.last_updated_by?.user?.username} {isWithin7Days(item.last_updated_by?.timestamp) ? formatDate(item.last_updated_by?.timestamp) : `at ${item.last_updated_by?.timestamp}`}</span>
                                ) : ( item.created && (<span>Created {isWithin7Days(item.created) ? formatDate(item.created) : `at ${item.created}`}</span>))}
                            </div>
                            { item.url && (
                                <div className="mb-6px">
                                    { item.url && (
                                        <a className="truncate w-full inline-block align-middle"  href={item.url} target="_blank" onClick={(e) => e.stopPropagation()}> 
                                            <SquareArrowOutUpRight className="relative top-2px w-14px h-14px"/> {item.url}
                                        </a>
                                    )}
                                </div>
                            )}
                        </List.Item>
                    </ResultDetail>
                )}
            />
        </div>
    )
}

export default memo(NormalList) ;
