import { Button, Modal } from 'antd';
import './Preview.css';
import { getServer } from '@/store/slice/server';
import normalizeUrl from 'normalize-url';

export const Preview = memo(props => {
  const { children, params = {}, widgetType, mode } = props;

  const { t } = useTranslation();

  const [isModalOpen, setIsModalOpen] = useState(false);

  const server = useAppSelector(getServer);

  const showModal = () => {
    setIsModalOpen(true);
  };

  const handleOk = () => {
    setIsModalOpen(false);
  };

  const handleCancel = () => {
    setIsModalOpen(false);
  };

  const htmlContent = useMemo(() => {
    if (!params.id || !widgetType) return ''
    return `
      <!DOCTYPE html>
      <html>
      <head>
        <title>Integration Preview</title>
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
        <style>
          html, body {
            padding: 0;
            margin: 0;
          }
        </style>
      </head>
      <body>
        <div id="${widgetType}" style="margin: ${mode === 'page' ? '0' : '10px'} 0; outline: none"></div>
        <script type="module" >
            import { ${widgetType} } from "${normalizeUrl(`${server}integration/${params.id}/widget`)}";
            ${widgetType}({container: "#${widgetType}", enableQueryParams: false });
        </script>
      </body>
      </html>
    `
  }, [params.id, widgetType, mode, server]);

  return (
    <>
      <span onClick={showModal}>{children}</span>
      <Modal
        destroyOnClose
        closeIcon={null}
        focusTriggerAfterClose={false}
        footer={null}
        keyboard={false}
        open={isModalOpen}
        title={null}
        wrapClassName="full-screen-modal"
        onCancel={handleCancel}
        onOk={handleOk}
      >
        <Button
          className="absolute right-12px top-12px"
          size="large"
          type="primary"
          onClick={handleCancel}
        >
          <SvgIcon
            className="text-18px"
            icon="mdi:exit-to-app"
          />{' '}
          {t('page.integration.code.exit')}
        </Button>
        <iframe
          height="100%"
          srcDoc={htmlContent}
          width="100%"
        />
      </Modal>
    </>
  );
});
