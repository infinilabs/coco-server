import { Typography } from "antd"
import { useState } from "react";

export const ErrorMessage = ({
    title,
    error,
}: {
    title: string;
    error?: string;
}) => {
    const [expanded, setExpanded] = useState(false);

    return (
        <div className="mt-16px px-12px rounded-8px border border-[#F0F0F0] dark:border-[#303030] ">
            <div className="h-38px leading-38px text-12px text-[#333] dark:text-[#E5E7EB] font-700">
                {title}
            </div>
            {
                error && (
                    <div className="py-8px min-h-42px border-t border-[#F0F0F0] dark:border-[#303030]">
                        <Typography.Paragraph
                            ellipsis={{
                                rows: 3,
                                expandable: 'collapsible',
                                expanded,
                                onExpand: (_, info) => setExpanded(info.expanded),
                            }}
                            className="!mb-0 leading-20px text-12px text-[#333] dark:text-[#E5E7EB]"
                        >
                            {error}
                        </Typography.Paragraph>
                    </div>
                )
            }
        </div>
    )
}