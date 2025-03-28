import { Modal } from "antd";
import "./Preview.css";

export const Preview = memo((props) => {
    const { children, params = {} } = props;

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
        <span onClick={showModal}>
            {children}
        </span>
        <Modal keyboard={false} destroyOnClose wrapClassName="full-screen-modal" title={null} footer={null} open={isModalOpen} onOk={handleOk} onCancel={handleCancel}>
            <iframe 
                src={`${window.location.origin}/widgets/searchbox.html?id=${params?.id}&token=${params?.token}&server=${window.location.origin}`}
                width={"100%"}
                height={"100%"}
            />
        </Modal>
        </>
    );
})