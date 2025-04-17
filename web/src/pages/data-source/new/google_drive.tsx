import { Button, Modal } from "antd";
import { ExclamationCircleOutlined } from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";

const { confirm } = Modal;

interface GoogleDriveProps {
  connector: {
    config?: {
      client_id?: string;
      client_secret?: string;
      auth_url?: string;
      redirect_url?: string;
      token_url?: string;
    };
  };
}

export default ({ connector }: GoogleDriveProps) => {
  const { t } = useTranslation();
  const nav = useNavigate();

  const onConnectClick = () => {
    const config = connector?.config || {};

    const missingFields = [
      config.client_id,
      config.client_secret,
      config.auth_url,
      config.redirect_url,
      config.token_url,
    ].some((field) => !field);

    if (missingFields) {
      confirm({
        title: t("common.tip"),
        icon: <ExclamationCircleOutlined />,
        content: t('page.datasource.missing_config_tip'),
        okText: t("common.confirm"),
        cancelText: t("common.cancel"),
        onOk() {
          nav("/connector/edit/google_drive", {state: connector });
        },
      });
    } else {
      window.location.href = `${window.location.origin}/connector/google_drive/connect`;
    }
  };

  return (
    <div className="flex items-center justify-between px-20px">
      <Button type="primary" onClick={onConnectClick}>
        {t("page.datasource.new.labels.connect")}
      </Button>
    </div>
  );
};
