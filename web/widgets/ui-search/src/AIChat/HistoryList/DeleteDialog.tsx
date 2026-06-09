import { Modal, Button } from "antd";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";

import { type Session } from "../types/chat";
import { type KeyboardEvent } from "react";

interface DeleteDialogProps {
  isOpen: boolean;
  active?: Session;
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

  const handleEnter = (event: KeyboardEvent, cb: () => void) => {
    if (event.code !== "Enter") return;

    event.stopPropagation();
    event.preventDefault();

    cb();
  };

  return (
    <Modal
      open={isOpen}
      onCancel={() => setIsOpen(false)}
      footer={null}
      width={360}
      title={t("history_list.delete_modal.title")}
    >
      <div className="flex flex-col justify-between">
        <div className="text-sm mb-4">
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
            >
              {t("history_list.delete_modal.button.cancel")}
            </Button>

          <Button
              danger
              type="primary"
              className="text-white"
              onClick={handleRemove}
              onKeyDown={(event) => {
                handleEnter(event, handleRemove);
              }}
            >
              {t("history_list.delete_modal.button.delete")}
            </Button>
        </div>
      </div>
    </Modal>
  );
};

export default DeleteDialog;
