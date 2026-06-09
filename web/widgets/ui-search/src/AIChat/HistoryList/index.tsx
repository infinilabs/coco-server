import { Input, Button, type InputRef } from "antd";
import { debounce } from "lodash";
import { type FC, useMemo, useRef, useState, type ChangeEvent } from "react";
import clsx from "clsx";
import { Search } from "lucide-react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";

import type { Session } from "../types/chat";
import HistoryListContent from "./HistoryListContent";
import RefreshIcon from "../../icons/RefreshIcon";

interface HistoryListProps {
  historyPanelId?: string;
  chats: Session[];
  active?: Session;
  onSearch: (keyword: string) => void;
  onRefresh: () => void;
  onSelect: (chat: Session) => void;
  onRename: (chatId: string, title: string) => void;
  onRemove: (chatId: string) => void;
  renamingId?: string;
  deletingId?: string;
  t?: TFunction;
}

const HistoryList: FC<HistoryListProps> = (props) => {
  const {
    historyPanelId,
    chats,
    active,
    onSearch,
    onRefresh,
    onSelect,
    onRename,
    onRemove,
    renamingId,
    deletingId,
    t: tProp,
  } = props;
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;
  const searchInputRef = useRef<InputRef>(null);
  const [isRefresh, setIsRefresh] = useState(false);
  const [keyword, setKeyword] = useState("");

  const filteredSessions = useMemo(() => {
    if (!keyword) return chats;
    
    return chats.filter(chat => {
      const title = (chat._source?.title || "") as string;
      return title.toLowerCase().includes(keyword.toLowerCase());
    });
  }, [chats, keyword]);

  const debouncedSearch = useMemo(() => {
    return debounce((value: string) => {
      setKeyword(value);
      onSearch(value);
    }, 300);
  }, [onSearch]);

  const handleRefresh = async () => {
    setIsRefresh(true);

    onRefresh();

    setTimeout(() => {
      setIsRefresh(false);
    }, 1000);
  };

  return (
    <div
      id={historyPanelId}
      className={clsx("flex flex-col h-full text-sm bg-transparent")}
    >
      <div className="flex gap-6px px-14px mt-16px">
        <div className="flex-1">
          <Input
            autoFocus
            ref={searchInputRef}
            prefix={
             <Search className="text-[#999] size-4" />
            }
            className="!px-8px w-full border-[#F0F0F0] dark:border-[#303030] rounded-12px"
            placeholder={t("history_list.search.placeholder")}
            onChange={(event: ChangeEvent<HTMLInputElement>) => {
              debouncedSearch(event.target.value);
            }}
          />
        </div>

        <Button
          type="default"
          className="size-8 p-0 flex items-center justify-center border-[#F0F0F0] dark:border-[#303030] rounded-12px"
          onClick={handleRefresh}
        >
          <RefreshIcon
            strokeWidth={2}
            className={clsx("size-4", {
              "animate-spin": isRefresh,
            })}
          />
        </Button>
      </div>

      <div className="flex-1 px-14px overflow-auto mt-8px">
        <HistoryListContent
          chats={filteredSessions}
          active={active}
          onSelect={onSelect}
          onRename={onRename}
          onRemove={onRemove}
          renamingId={renamingId}
          deletingId={deletingId}
          t={t}
        />
      </div>
    </div>
  );
};

export default HistoryList;
