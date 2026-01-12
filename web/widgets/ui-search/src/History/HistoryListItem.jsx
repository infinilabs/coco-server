import { useState, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { MoreHorizontal, Pencil, Trash2 } from 'lucide-react';
import { Dropdown, Input } from 'antd';
import clsx from 'clsx';

const HistoryListItem = (props) => {
  const {
    item,
    active,
    onSelect,
    onRename,
    onDelete,
    onMouseEnter,
    highlightId
  } = props;
  const { t } = useTranslation();
  const { _id, _source } = item;
  const title = _source?.title || _source?.message || _id;

  const isSelected = active && (typeof active === 'string' ? item._id === active : item._id === active._id);
  const isHovered = item._id === highlightId;

  const [isEdit, setIsEdit] = useState(false);
  const [editTitle, setEditTitle] = useState(title);
  const inputRef = useRef(null);

  useEffect(() => {
    if (isEdit && inputRef.current) {
        inputRef.current.focus();
    }
  }, [isEdit]);

  const handleRenameClick = () => {
    setEditTitle(title);
    setIsEdit(true);
  };

  const handleRenameSubmit = () => {
    if (editTitle && editTitle.trim() !== "") {
        onRename(_id, editTitle);
    }
    setIsEdit(false);
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter') {
        handleRenameSubmit();
    } else if (e.key === 'Escape') {
        setIsEdit(false);
        setEditTitle(title);
    }
  };

  const menuItems = [
    {
      key: 'rename',
      label: (
        <div className="flex items-center gap-2">
            <Pencil className="w-4 h-4" />
            <span>{t("history_list.menu.rename", "Rename")}</span>
        </div>
      ),
      onClick: ({ domEvent }) => {
        domEvent.stopPropagation();
        handleRenameClick();
      }
    },
    {
      key: 'delete',
      danger: true,
      label: (
        <div className="flex items-center gap-2">
            <Trash2 className="w-4 h-4" />
            <span>{t("history_list.menu.delete", "Delete")}</span>
        </div>
      ),
      onClick: ({ domEvent }) => {
        domEvent.stopPropagation();
        onDelete();
      }
    },
  ];

  if (isEdit) {
    return (
        <div className="px-2 py-1">
            <Input
                ref={inputRef}
                value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
                onBlur={handleRenameSubmit}
                onKeyDown={handleKeyDown}
                size="small"
                onClick={(e) => e.stopPropagation()}
            />
        </div>
    );
  }

  return (
    <li
      id={_id}
      className={clsx(
        "group flex w-full items-center mt-1 px-2 py-2 rounded-lg cursor-pointer transition-colors relative",
        {
          "bg-[#E5E7EB] dark:bg-[#2B3444]": isSelected,
          "hover:bg-[#EDEDED] dark:hover:bg-[#353F4D]": !isSelected,
        }
      )}
      onClick={() => onSelect(item)}
      onMouseEnter={onMouseEnter}
    >
      <div className="flex-1 truncate text-sm text-[#333] dark:text-[#d8d8d8] pr-6">
        {title}
      </div>

      {(isHovered || isSelected) && (
        <div className="absolute right-2 top-1/2 transform -translate-y-1/2" onClick={(e) => e.stopPropagation()}>
            <Dropdown menu={{ items: menuItems }} trigger={['click']}>
                <div className="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-600 cursor-pointer">
                    <MoreHorizontal className="w-4 h-4 text-gray-500" />
                </div>
            </Dropdown>
        </div>
      )}
    </li>
  );
};

export default HistoryListItem;
