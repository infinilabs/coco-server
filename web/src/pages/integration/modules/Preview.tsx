import { Button, Modal } from "antd";
import "./Preview.css";

export const Preview = memo((props) => {
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

    params.server = "http://localhost:9001"

    return (
        <>
        <span onClick={showModal}>
            {children}
        </span>
        <Modal 
            keyboard={false} 
            destroyOnClose 
            wrapClassName="full-screen-modal" 
            title={null} 
            footer={null} 
            open={isModalOpen} 
            onOk={handleOk} 
            onCancel={handleCancel}
            closeIcon={null}
            focusTriggerAfterClose={false}
        >
            <Button onClick={handleCancel} size="large" type="primary" className="absolute right-12px top-12px">
                <SvgIcon className="text-18px" icon="mdi:exit-to-app"/> {t('page.integration.code.exit')}
            </Button>
            <iframe 
                src={`${params.server}/widgets/searchbox/index.html?id=${params?.id}&token=${params?.token}&server=${encodeURIComponent(params.server)}`}
                width={"100%"}
                height={"100%"}
            />
        </Modal>
        </>
    );
})