import { Tag } from "antd";
import { Lightbulb } from "lucide-react";
import ListContainer from "./ListContainer";

export const SUGGESTION_TIPS = "suggestion_tips"

export default (props) => {
    const { } = props;

    return (
        <ListContainer
            type={SUGGESTION_TIPS}
            title="搜索 Tips"
            data={[
                {
                    icon: <Lightbulb className="w-16px h-16px" />,
                    suggestion: <span>按 <Tag>/</Tag> 启用高级字段过滤，或直接输入 <Tag>字段名</Tag> + <Tag>:</Tag> 转为条件</span>,
                },
            ]}
            defaultRows={1}
        />
    )
};