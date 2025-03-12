import { Button } from "antd";

export default () => {
  const { t } = useTranslation();
  return (
    <div className="flex items-center justify-between px-20px">
      <Button type="primary">
        <a href={window.location.origin + "/connector/google_drive/connect"}>
          {t("page.datasource.new.labels.connect")}
        </a>
      </Button>
    </div>
  );
};
