import { useState, useMemo, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import groupBy from "lodash/groupBy";
import isNil from "lodash/isNil";
import dayjs from 'dayjs';
import isSameOrAfter from 'dayjs/plugin/isSameOrAfter';
import HistoryListItem from './HistoryListItem';
import DeleteDialog from './DeleteDialog';

dayjs.extend(isSameOrAfter);

const HistoryListContent = (props) => {
  const {
    chats,
    active,
    onSelect,
    onRename,
    onRemove,
  } = props;
  const { t } = useTranslation();
  const listRef = useRef(null);

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [highlightId, setHighlightId] = useState("");
  const [itemToDelete, setItemToDelete] = useState(null);

  const sortedList = useMemo(() => {
    if (isNil(chats) || chats.length === 0) return {};

    const now = dayjs();

    return groupBy(chats, (chat) => {
      const date = dayjs(chat._source?.created);

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

  const handleDeleteRequest = (chat) => {
    setItemToDelete(chat);
    setDeleteDialogOpen(true);
  };

  const confirmDelete = () => {
    if (itemToDelete) {
        onRemove(itemToDelete._id);
        setDeleteDialogOpen(false);
        setItemToDelete(null);
    }
  };

  const hasChats = chats && chats.length > 0;

  return (
    <>
      <div ref={listRef} className="pb-4">
        {!hasChats && (
             <div className="flex flex-col items-center justify-center h-40 text-gray-400">
                <span className="text-sm">{t("history_list.no_history", "No history")}</span>
             </div>
        )}
        {Object.entries(sortedList).map(([dateLabel, groupChats]) => (
          <div key={dateLabel} className="mb-4">
            <div className="sticky top-0 z-10 py-2 px-2 text-xs font-medium text-gray-500 bg-[#F3F4F6] dark:bg-[#1F2937]">
              {dateLabel.startsWith('history_list.date.') ? t(dateLabel) : dateLabel}
            </div>
            <ul>
              {groupChats.map((chat) => (
                <HistoryListItem
                  key={chat._id}
                  item={chat}
                  active={active}
                  onSelect={onSelect}
                  onRename={onRename}
                  onDelete={() => handleDeleteRequest(chat)}
                  onMouseEnter={() => setHighlightId(chat._id)}
                  highlightId={highlightId}
                />
              ))}
            </ul>
          </div>
        ))}
      </div>

      <DeleteDialog 
        isOpen={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        onConfirm={confirmDelete}
        item={itemToDelete}
      />
    </>
  );
};

export default HistoryListContent;
