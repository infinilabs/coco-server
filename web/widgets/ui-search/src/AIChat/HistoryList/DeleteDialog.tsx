import { Modal, Button } from "antd";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";

import { type Chat } from "../types/chat";
import { type KeyboardEvent, useRef } from "react";

interface DeleteDialogProps {
  isOpen: boolean;
  active?: Chat;
  setIsOpen: (isOpen: boolean) => void;
  handleRemove: () => void;
  t?: TFunction;
}

const DeleteDialog = ({
  isOpen,
  active,
  setIsOpen,
  handleRemove,
  t: tProp,
}: DeleteDialogProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;
  const wrapperRef = useRef<HTMLDivElement>(null);

  const handleEnter = (event: KeyboardEvent, cb: () => void) => {
    if (event.code !== "Enter") return;

    event.stopPropagation();
    event.preventDefault();

    cb();
  };

  return (
    <div ref={wrapperRef} style={{ display: 'contents' }}>
      <Modal
        open={isOpen}
        onCancel={() => setIsOpen(false)}
        footer={null}
        width={480}
        title={(
          <div className="text-16px text-[#333] dark:text-[#E5E7EB]">
            {t("history_list.delete_modal.title")}
          </div>
        )}
        destroyOnHidden
        classNames={{
          container: "!p-24px",
          header: "!mb-24px",
        }}
        getContainer={() => {
          return wrapperRef.current?.closest?.('.ui-search') as HTMLElement || document.body;
        }}
      >
        <div className="flex flex-col justify-between">
          <div className="min-h-52px text-16px text-[#666] dark:text-white/80 mb-20px">
            {t("history_list.delete_modal.description", {
              item:
                (active?._source?.title as string) ||
                (active?._source?.message as string) ||
                active?._id,
            })}
          </div>

          <div className="flex gap-4 self-end">
            <Button
              autoFocus
              onClick={() => setIsOpen(false)}
              onKeyDown={(event) => {
                handleEnter(event, () => {
                  setIsOpen(false);
                });
              }}
              className="w-87px text-12px text-[#333] dark:text-[#E5E7EB] rounded-20px border-[#bbb] dark:border-[#333]"
            >
              {t("history_list.delete_modal.button.cancel")}
            </Button>

            <Button
              color="danger"
              variant="outlined"
              onClick={handleRemove}
              onKeyDown={(event) => {
                handleEnter(event, handleRemove);
              }}
              className="w-87px text-12px rounded-20px"
            >
              {t("history_list.delete_modal.button.delete")}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default DeleteDialog;
