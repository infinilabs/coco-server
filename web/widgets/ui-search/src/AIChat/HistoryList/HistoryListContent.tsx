import { type FC, useCallback, useState, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import { groupBy, isNil } from "lodash";
import dayjs from "dayjs";
import isSameOrAfter from "dayjs/plugin/isSameOrAfter";

import type { Session } from "../types/chat";
import NoDataImage from "../NoDataImage";
import DeleteDialog from "./DeleteDialog";
import HistoryListItem from "./HistoryListItem";

dayjs.extend(isSameOrAfter);

interface HistoryListContentProps {
  chats: Session[];
  active?: Session;
  onSelect: (chat: Session) => void;
  onRename: (chatId: string, title: string) => void;
  onRemove: (chatId: string) => void;
  renamingId?: string;
  deletingId?: string;
  t?: TFunction;
}

const HistoryListContent: FC<HistoryListContentProps> = ({
  chats,
  active,
  onSelect,
  onRename,
  onRemove,
  renamingId,
  deletingId,
  t: tProp,
}) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;

  const [isOpen, setIsOpen] = useState(false);
  const [highlightItem, setHighlightItem] = useState<Session>({} as Session);

  const sortedList = useMemo(() => {
    if (isNil(chats)) return {};

    const now = dayjs();

    return groupBy(chats, (chat) => {
      const date = dayjs(chat._source?.created as unknown as string | number | Date | undefined);

      if (date.isSame(now, "day")) {
        return "history_list.date.today";
      }

      if (date.isSame(now.subtract(1, "day"), "day")) {
        return "history_list.date.yesterday";
      }

      if (date.isSameOrAfter(now.subtract(7, "day"), "day")) {
        return "history_list.date.last7Days";
      }

      if (date.isSameOrAfter(now.subtract(30, "day"), "day")) {
        return "history_list.date.last30Days";
      }

      return date.format("YYYY-MM");
    });
  }, [chats]);

  const handleRemove = () => {
    if (!highlightItem?._id) return;

    onRemove(highlightItem._id);

    setIsOpen(false);
  };

  const handleDelete = useCallback((chat: Session) => {
    setHighlightItem(chat);
    setIsOpen(true);
  }, []);

  if (chats.length === 0) {
    return (
      <div className="flex items-center justify-center flex-1" style={{ paddingTop: "32px" }}>
        <NoDataImage />
      </div>
    );
  }

  return (
    <>
      <div className="py-4">
        {Object.entries(sortedList).map(([label, list]) => (
          <div key={label}>
            <div className="text-14px text-[#999] py-8px">{t(label)}</div>
            <ul className="p-0">
              {list.map((item) => (
                <HistoryListItem
                  key={item._id}
                  item={item}
                  active={active}
                  onSelect={onSelect}
                  onRename={onRename}
                  handleDelete={() => handleDelete(item)}
                  renamingId={renamingId}
                  deletingId={deletingId}
                  t={t}
                />
              ))}
            </ul>
          </div>
        ))}
      </div>

      <DeleteDialog
        isOpen={isOpen}
        active={active || highlightItem}
        setIsOpen={setIsOpen}
        handleRemove={handleRemove}
        t={t}
      />
    </>
  );
};

export default HistoryListContent;
