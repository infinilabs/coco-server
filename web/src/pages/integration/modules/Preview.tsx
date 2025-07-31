import { Button, Modal } from 'antd';
import './Preview.css';

export const Preview = memo(props => {
  const { children, params = {} } = props;

  const { t } = useTranslation();

  const [isModalOpen, setIsModalOpen] = useState(false);

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
    if (!params.id || !params.type) return ''
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
        <div id="${params.type}" style="margin: 10px 0; outline: none"></div>
        <script type="module" >
            import { ${params.type} } from "${window.location.origin}/integration/${params.id}/widget";
            ${params.type}({container: "#${params.type}"});
        </script>
      </body>
      </html>
    `
  }, [params.id, params.type]);

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
          // src={`${params.server}/widgets/${params.type}/index.html?id=${params?.id}&token=${params?.token}&server=${encodeURIComponent(params.server)}`}
          width="100%"
        />
      </Modal>
    </>
  );
});
