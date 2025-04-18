import { DEFAULT_PAGE_SIZE } from ".";
import List from "./List";
import { useMemo } from "react";

const CustomList = (props) => {
    const { visible, data = [], pagination = {} } = props;

    const total = pagination.total || data.length;
    const pageSize = pagination.pageSize || DEFAULT_PAGE_SIZE

    return (
        <List 
            {...props}  
            visible={visible}
            data={data}
            pagination={{
                ...pagination,
                pageSize,
                pages: Math.ceil(total / pageSize)
            }}
        />
    )
}

export default CustomList 