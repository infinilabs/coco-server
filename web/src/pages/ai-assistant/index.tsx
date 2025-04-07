import { Button, Card, Result } from "antd";

export function Component() {

  const { t } = useTranslation();
  const nav = useNavigate();

  const onClick = () => {
    nav('/');
  };

  return (
    <div className="h-full min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
      <ACard
        bordered={false}
        className="card-wrapper h-100% flex items-center"
      >
        <Result
          title={t('common.comingSoon')}
          extra={
            <Button
              type="primary"
              onClick={onClick}
            >
              {t('common.backToHome')}
            </Button>
          }
        />
      </ACard>
    </div>
  );
}
