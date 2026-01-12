import { Modal, Button } from 'antd';
import { useTranslation } from 'react-i18next';

const DeleteDialog = (props) => {
  const { isOpen, onClose, onConfirm, item } = props;
  const { t } = useTranslation();

  const title = item?._source?.title || item?._source?.message || item?._id;

  return (
    <Modal
      title={t("history_list.delete_modal.title", "Delete Chat")}
      open={isOpen}
      onCancel={onClose}
      onOk={onConfirm}
      okText={t("history_list.delete_modal.button.delete", "Delete")}
      cancelText={t("history_list.delete_modal.button.cancel", "Cancel")}
      okButtonProps={{ danger: true }}
      centered
    >
      <p>
        {t("history_list.delete_modal.description", {
            defaultValue: `Are you sure you want to delete "${title}"? This action cannot be undone.`,
            title: title
        })}
      </p>
    </Modal>
  );
};

export default DeleteDialog;
