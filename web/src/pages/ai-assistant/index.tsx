import { Button, Card, Result } from "antd";

export function Component() {

  const { t } = useTranslation();
  const nav = useNavigate();

  const onClick = () => {
    nav('/');
  };

  return (
    <Card className="h-100% flex items-center">
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
    </Card>
  );
}
