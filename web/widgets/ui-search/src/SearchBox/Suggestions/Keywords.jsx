import { MessageCircle, Search } from "lucide-react";
import ListContainer from "./ListContainer";
import { useState, useEffect, useRef } from "react";

export const SUGGESTION_ACTIONS = "suggestion_actions"
export const SUGGESTION_KEYWORDS = "suggestion_keywords"

export default (props) => {
    const { keyword, data = [], onItemSelect, onItemClick } = props;

    // 定义 actions 数据
    const actions = [
        {
            action: "search",
            icon: <Search className="w-16px h-16px" />,
            suggestion: keyword,
            source: `快速查找 | 直达文件与结果`,
        },
        {
            action: "deepthink",
            icon: <MessageCircle className="w-16px h-16px" />,
            suggestion: keyword,
            source: `深度思考 | AI 提炼，结论优先`,
        },
        {
            action: "deepresearch",
            icon: <Search className="w-16px h-16px" />,
            suggestion: keyword,
            source: `深度研究 | 多步推理，综合分析`,
        },
    ].filter(item => !!item?.suggestion); // 过滤无效项

    // 过滤后的 keywords 数据
    const keywords = data.filter(item => !!item?.suggestion);

    // 合并所有数据（actions + keywords）
    const combinedData = useRef([...actions, ...keywords]);
    // 记录 actions 数据长度，用于区分两个列表
    const actionsLength = actions.length;

    // 全局选中索引
    const [globalActiveIndex, setGlobalActiveIndex] = useState(0);
    // 子组件引用，用于触发点击事件
    const listRefs = useRef({
        [SUGGESTION_ACTIONS]: null,
        [SUGGESTION_KEYWORDS]: null
    });

    // 更新合并数据
    useEffect(() => {
        combinedData.current = [...actions, ...keywords];
        // 如果选中索引超出新数据长度，重置为 -1
        if (globalActiveIndex >= combinedData.current.length) {
            setGlobalActiveIndex(-1);
        }
    }, [actions, keywords]);

    // 统一的键盘事件处理（仅在有多个列表时启用）
    useEffect(() => {
        // 只有当存在多个有效列表时才注册全局键盘事件
        const hasMultipleLists = (actions.length > 0 && keywords.length > 0);
        if (!hasMultipleLists || !onItemClick) return;

        const handleKeyDown = (e) => {
            if (![38, 40, 13].includes(e.keyCode)) return;

            const totalItems = combinedData.current.length;
            if (totalItems === 0) return;

            e.preventDefault();
            let newIndex = -1
            switch (e.keyCode) {
                case 40: // 下键
                    newIndex = globalActiveIndex === -1 ? 0 : globalActiveIndex + 1;
                    // 限制最大索引为 合并数据长度 - 1
                    if (newIndex >= combinedData.current.length) {
                        newIndex = combinedData.current.length - 1;
                    }
                    // ... 其他逻辑
                    setGlobalActiveIndex(newIndex);
                    onItemSelect(combinedData.current?.[newIndex])
                    break;

                case 38: // 上键
                    if (globalActiveIndex === -1) return;
                    newIndex = globalActiveIndex - 1;
                    // 限制最小索引为 0
                    if (newIndex < 0) {
                        newIndex = 0;
                    }
                    // ... 其他逻辑
                    setGlobalActiveIndex(newIndex);
                    onItemSelect(combinedData.current?.[newIndex])
                    break;
                case 13: // 回车
                    if (globalActiveIndex >= 0 && globalActiveIndex < totalItems) {
                        const item = combinedData.current[globalActiveIndex];
                        // 触发点击事件
                        onItemClick?.(item);
                        // 通知对应子组件触发点击
                        if (globalActiveIndex < actionsLength) {
                            listRefs.current[SUGGESTION_ACTIONS]?.triggerItemClick(globalActiveIndex);
                        } else {
                            listRefs.current[SUGGESTION_KEYWORDS]?.triggerItemClick(globalActiveIndex - actionsLength);
                        }
                    }
                    break;
                default:
                    break;
            }
        };

        document.addEventListener("keydown", handleKeyDown);

        return () => {
            document.removeEventListener("keydown", handleKeyDown);
        };
    }, [globalActiveIndex, onItemClick, actionsLength, actions.length, keywords.length]);

    // 子组件回调：设置引用和获取相对索引
    const setListRef = (type, ref) => {
        listRefs.current[type] = ref;
    };

    // 计算子组件的本地激活索引
    const getLocalActiveIndex = (type) => {
        // 只有存在多个列表时才使用全局索引
        const hasMultipleLists = (actions.length > 0 && keywords.length > 0);
        if (!hasMultipleLists) return -1;

        if (globalActiveIndex === -1) return -1;

        if (type === SUGGESTION_ACTIONS) {
            // actions 列表：索引在 actions 范围内才有效
            return globalActiveIndex < actionsLength ? globalActiveIndex : -1;
        } else {
            // keywords 列表：索引转换为本地索引
            const localIndex = globalActiveIndex - actionsLength;
            return localIndex >= 0 && localIndex < keywords.length ? localIndex : -1;
        }
    };

    // 判断是否启用全局键盘模式
    const useGlobalKeydown = (actions.length > 0 && keywords.length > 0);

    return (
        <>
            <ListContainer
                {...props}
                className={data.length !== 0 ? "!mb-0" : ""}
                ref={el => setListRef(SUGGESTION_ACTIONS, el)}
                type={SUGGESTION_ACTIONS}
                data={actions}
                useGlobalKeydown={useGlobalKeydown}
                globalActiveIndex={getLocalActiveIndex(SUGGESTION_ACTIONS)}
                onGlobalSelect={(localIndex) => {
                    setGlobalActiveIndex(localIndex);
                }}
            />
            <ListContainer
                {...props}
                ref={el => setListRef(SUGGESTION_KEYWORDS, el)}
                type={SUGGESTION_KEYWORDS}
                title="搜索建议"
                data={data}
                useGlobalKeydown={useGlobalKeydown}
                globalActiveIndex={getLocalActiveIndex(SUGGESTION_KEYWORDS)}
                onGlobalSelect={(localIndex) => {
                    setGlobalActiveIndex(actionsLength + localIndex);
                }}
            />
        </>
    );
};