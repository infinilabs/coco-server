import {
  type FC,
  useRef,
  useCallback,
  useState,
  useEffect,
  type KeyboardEvent,
} from "react";
import { Ellipsis, Loader2 } from "lucide-react";
import clsx from "clsx";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import { Pencil, Trash2 } from "lucide-react";
import { Popover, Input } from "antd";

import type { Chat } from "../types/chat";

interface HistoryListItemProps {
  item: Chat;
  active?: Chat;
  onSelect: (chat: Chat) => void;
  onRename: (chatId: string, title: string) => void;
  handleDelete: () => void;
  renamingId?: string;
  deletingId?: string;
  t?: TFunction;
}

const HistoryListItem: FC<HistoryListItemProps> = ({
  item,
  active,
  onSelect,
  onRename,
  handleDelete,
  renamingId,
  deletingId,
  t: tProp,
}) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;
  const moreButtonRef = useRef<HTMLButtonElement>(null);
  const { _id, _source } = item;
  const title = (_source?.title ?? _id) as string;
  const isSelected = item._id === active?._id;
  const isRenaming = renamingId === item._id;
  const isDeleting = deletingId === item._id;

  const [isEdit, setIsEdit] = useState(false);
  const [open, setOpen] = useState(false);
  const [isHovered, setIsHovered] = useState(false);

  const onContextMenu = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();

      if (isSelected) {
        moreButtonRef.current?.click();
      }
    },
    [moreButtonRef, isSelected]
  );

  const handleRename = useCallback(() => {
    setIsEdit(true);
  }, []);

  const menuItems = [
    // {
    //   label: "history_list.menu.share",
    //   icon: Share2,
    //   onClick: () => {},
    // },
    {
      label: "history_list.menu.rename",
      icon: Pencil,
      shortcut: "R",
      onClick: handleRename,
    },
    {
      label: "history_list.menu.delete",
      icon: Trash2,
      shortcut: "D",
      iconColor: "#FF2018",
      onClick: handleDelete,
    },
  ];

  useEffect(() => {
    if (open && isEdit) {
      requestAnimationFrame(() => setOpen(false));
    }
  }, [isEdit, open]);

  const popoverContent = (
    <div
      className="flex flex-col gap-2"
      onClick={(event) => {
        event.stopPropagation();
      }}
    >
      {menuItems.map((menuItem) => {
        const { label, icon: Icon, iconColor, onClick } = menuItem;

        return (
          <button
            key={label}
            className="flex items-center gap-2 py-6px px-8px text-sm rounded-md bg-transparent hover:bg-[#EDEDED] dark:hover:bg-[#353F4D] border-none transition text-left cursor-pointer"
            onClick={() => {
              onClick();
              setOpen(false);
            }}
          >
            <Icon
              className="size-4"
              style={{
                color: iconColor,
              }}
            />

            <span>{t(label)}</span>
          </button>
        );
      })}
    </div>
  );

  return (
    <li
      key={_id}
      id={_id}
      className={clsx(
        "group flex w-full items-center h-36px rounded-lg cursor-pointer transition-colors relative",
        {
          "bg-[#E5E7EB] dark:bg-[#2B3444]": isSelected,
          "hover:bg-[#EDEDED] dark:hover:bg-[#353F4D]": !isSelected,
        }
      )}
      onClick={() => {
        if (!isSelected) {
          setIsEdit(false);
        }

        onSelect(item);
      }}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      onContextMenu={onContextMenu}
    >
      <div
        className={clsx("w-1 h-6 rounded-sm bg-[#0072FF]", {
          "opacity-0": item._id !== active?._id,
        })}
      />

      <div className="flex-1 flex items-center justify-between gap-2 px-2 overflow-hidden">
        {isEdit || isRenaming ? (
          <Input
            autoFocus
            disabled={isRenaming}
            defaultValue={title}
            className="flex-1 -mx-px h-7"
            suffix={
              isRenaming ? (
                <span className="flex items-center">
                  <Loader2 className="size-4 animate-spin" />
                </span>
              ) : undefined
            }
            onKeyDown={(event: KeyboardEvent<HTMLInputElement>) => {
              if (isRenaming) return;
              if (event.key === "Escape") {
                event.stopPropagation();
                setIsEdit(false);
                return;
              }
              if (event.key !== "Enter") return;

              event.stopPropagation();

              const value = event.currentTarget.value;
              onRename(item._id || "", value);
              setIsEdit(false);
            }}
            onBlur={() => {
              if (isRenaming) return;
              setIsEdit(false);
            }}
          />
        ) : (
          <span className="truncate">{title}</span>
        )}

        {(isHovered || isDeleting || isSelected || open) && !(isEdit || isRenaming) && (<div
          className="flex items-center gap-2"
          onClick={(e) => {
            e.stopPropagation();
          }}
        >
          {isDeleting ? (
            <Loader2 className="size-4 animate-spin text-muted-foreground" />
          ) : (
            <Popover
              open={open}
              onOpenChange={setOpen}
              content={popoverContent}
              trigger="click"
              placement="bottomRight"
              arrow={false}
              getPopupContainer={(node) => {
                let el = node.parentElement;
                while (el) {
                  const { overflowY } = getComputedStyle(el);
                  if (overflowY === "auto" || overflowY === "scroll") {
                    return el.parentElement || el;
                  }
                  el = el.parentElement;
                }
                return node.parentElement || document.body;
              }}
              classNames={{
                container: "!px-4px !py-8px",
              }}
            >
              <button
                ref={moreButtonRef}
                className={clsx(
                  "flex gap-2 bg-transparent border-none p-0 cursor-pointer outline-none",
                  {
                    "opacity-100":
                      open || (!isEdit && isSelected),
                    "opacity-0 group-hover:opacity-100": !(
                      open ||
                      (!isEdit && isSelected)
                    ),
                  }
                )}
              >
                <Ellipsis className="size-4 text-[#979797]" />
              </button>
            </Popover>
          )}
        </div>)}
      </div>
    </li>
  );
};

export default HistoryListItem;
