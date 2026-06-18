import { MessageCircle, Search } from "lucide-react";
import ListContainer from "./ListContainer";
import { useState, useEffect, useRef, useMemo, type FC, type RefObject } from "react";
import { useTranslation } from 'react-i18next';
import DeepresearchIcon from "../../icons/DeepresearchIcon";

export const SUGGESTION_ACTIONS = "suggestion_actions"
export const SUGGESTION_KEYWORDS = "suggestion_keywords"

interface KeywordsProps {
    keyword?: string;
    data?: any[];
    onItemSelect?: (item: any) => void;
    onItemClick?: (item: any) => void;
    action_type?: string;
    settings?: Record<string, any>;
    language?: string;
    keyboardRootRef?: RefObject<HTMLElement | null>;
}

const Keywords: FC<KeywordsProps> = (props) => {
    const { keyword, data = [], onItemSelect, onItemClick, action_type, settings, keyboardRootRef } = props;
    const { t } = useTranslation();

    const actions = useMemo(() => [
        {
            action: "search",
            icon: <Search className="w-16px h-16px" />,
            suggestion: keyword,
            source: t('labels.quickFind'),
        },
        settings?.deep_think_assistant_entity?.type === 'deep_think' ? {
            action: "deepthink",
            icon: <MessageCircle className="w-16px h-16px" />,
            suggestion: keyword,
            source: t('labels.deepThinkShort'),
            assistant_id: settings?.deep_think_assistant_entity?.id,
        } : null,
        settings?.deep_research_assistant_entity?.type === 'deep_research' ? {
            action: "deepresearch",
            icon: <DeepresearchIcon className="w-16px h-16px" />,
            suggestion: keyword,
            source: t('labels.deepResearchShort'),
            assistant_id: settings?.deep_research_assistant_entity?.id,
        } : null,
    ].filter(item => !!item?.suggestion), [keyword, t, settings]);

    const keywords = useMemo(() => data.filter(item => !!item?.suggestion), [data]);

    const combinedData = useRef<any[]>([]);
    const actionsLength = actions.length;

    const [globalActiveIndex, setGlobalActiveIndex] = useState(() => {
        if (action_type) {
            const index = actions.findIndex(item => item?.action === action_type);
            return index >= 0 ? index : 0;
        }
        return 0;
    });

    const listRefs = useRef<Record<string, any>>({
        [SUGGESTION_ACTIONS]: null,
        [SUGGESTION_KEYWORDS]: null
    });
    const skipActionTypeResetRef = useRef(false);

    useEffect(() => {
        combinedData.current = [...actions, ...keywords];
        if (action_type) {
            if (skipActionTypeResetRef.current) {
                skipActionTypeResetRef.current = false;
                return;
            }
            const index = actions.findIndex(item => item?.action === action_type);
            if (index >= 0) {
                setGlobalActiveIndex(index);
                onItemSelect?.(combinedData.current?.[index])
                return;
            }
        }
        if (globalActiveIndex >= combinedData.current.length) {
            setGlobalActiveIndex(-1);
        }
    }, [actions, keywords, action_type]);

    useEffect(() => {
        const hasMultipleLists = (actions.length > 0 && keywords.length > 0);
        if (!hasMultipleLists || !onItemClick) return;
        const keyboardTarget = keyboardRootRef?.current;
        if (!keyboardTarget) return;

        const handleKeyDown = (e: KeyboardEvent) => {
            if (![38, 40, 13].includes(e.keyCode)) return;

            const totalItems = combinedData.current.length;
            if (totalItems === 0) return;

            e.preventDefault();
            let newIndex = -1
            switch (e.keyCode) {
                case 40:
                    newIndex = globalActiveIndex === -1 ? 0 : globalActiveIndex + 1;
                    if (newIndex >= combinedData.current.length) {
                        newIndex = combinedData.current.length - 1;
                    }
                    setGlobalActiveIndex(newIndex);
                    if (newIndex < actionsLength) {
                        onItemSelect?.(combinedData.current?.[newIndex]);
                    } else {
                        skipActionTypeResetRef.current = true;
                        onItemSelect?.({ ...(combinedData.current?.[newIndex] || {}), action: "search" });
                    }
                    break;

                case 38: 
                    if (globalActiveIndex === -1) return;
                    newIndex = globalActiveIndex - 1;
                    if (newIndex < 0) {
                        newIndex = 0;
                    }
                    setGlobalActiveIndex(newIndex);
                    if (newIndex < actionsLength) {
                        onItemSelect?.(combinedData.current?.[newIndex]);
                    } else {
                        skipActionTypeResetRef.current = true;
                        onItemSelect?.({ ...(combinedData.current?.[newIndex] || {}), action: "search" });
                    }
                    break;
                case 13: 
                    if (globalActiveIndex >= 0 && globalActiveIndex < totalItems) {
                        const item = combinedData.current[globalActiveIndex];
                        onItemClick?.(item);
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

        keyboardTarget.addEventListener("keydown", handleKeyDown);

        return () => {
            keyboardTarget.removeEventListener("keydown", handleKeyDown);
        };
    }, [globalActiveIndex, onItemClick, actionsLength, actions.length, keywords.length, keyboardRootRef]);

    const setListRef = (type: string, ref: any) => {
        listRefs.current[type] = ref;
    };

    const getLocalActiveIndex = (type: string) => {
        const hasMultipleLists = (actions.length > 0 && keywords.length > 0);
        if (!hasMultipleLists) return -1;

        if (globalActiveIndex === -1) return -1;

        if (type === SUGGESTION_ACTIONS) {
            return globalActiveIndex < actionsLength ? globalActiveIndex : -1;
        } else {
            const localIndex = globalActiveIndex - actionsLength;
            return localIndex >= 0 && localIndex < keywords.length ? localIndex : -1;
        }
    };

    const useGlobalKeydown = (actions.length > 0 && keywords.length > 0);

    const actionsDefaultActiveIndex = useMemo(() => {
        if (action_type) {
            const index = actions.findIndex(item => item?.action === action_type);
            return index >= 0 ? index : 0;
        }
        return 0;
    }, [action_type, actions]);

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
                defaultActiveIndex={actionsDefaultActiveIndex}
                onGlobalSelect={(localIndex) => {
                    setGlobalActiveIndex(localIndex);
                    onItemSelect?.(combinedData.current?.[localIndex]);
                }}
            />
            <ListContainer
                {...props}
                ref={el => setListRef(SUGGESTION_KEYWORDS, el)}
                type={SUGGESTION_KEYWORDS}
                title={t('labels.suggestionsTitle')}
                data={data.map((item) => ({
                    ...item,
                    icon: <Search className="w-16px h-16px" />
                }))}
                useGlobalKeydown={useGlobalKeydown}
                globalActiveIndex={getLocalActiveIndex(SUGGESTION_KEYWORDS)}
                onGlobalSelect={(localIndex) => {
                    const globalIndex = actionsLength + localIndex;
                    setGlobalActiveIndex(globalIndex);
                    skipActionTypeResetRef.current = true;
                    onItemSelect?.({ ...(combinedData.current?.[globalIndex] || {}), action: "search" });
                }}
            />
        </>
    );
};

export default Keywords;