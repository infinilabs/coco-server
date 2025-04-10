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
          src={`${params.server}/widgets/searchbox/index.html?id=${params?.id}&token=${params?.token}&server=${encodeURIComponent(params.server)}`}
          width="100%"
        />
      </Modal>
    </>
  );
});
