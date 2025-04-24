import React, {
  forwardRef,
  useImperativeHandle,
  useState,
  useRef,
} from "react";
import { Tabs, Modal } from "antd";
import styles from "./index.module.less";
import Version from "./Version";

export const DATE_FORMAT = "YYYY.MM.DD HH:mm";

export default forwardRef((props, ref) => {
  const [visible, setVisible] = useState(false);
  const tabRef = useRef(null);
  const { t } = useTranslation();

  const tabs = [
    {
      key: "version",
      title: t("license.title"),
      component: Version,
    },
  ];

  useImperativeHandle(ref, () => ({
    open: onOpen,
    close: onClose,
  }));

  const onOpen = () => {
    setVisible(true);
  };

  const onClose = () => {
    setVisible();
  };

  return (
    <Modal
      open={visible}
      wrapClassName={styles.systemLicense}
      closable
      footer={null}
      onCancel={onClose}
      destroyOnClose
      width={580}
    >
      <Tabs defaultActiveKey="version">
        {tabs.map((item) => (
          <Tabs.TabPane tab={<div className="px-12px">{item.title}</div>} key={item.key}>
            <div className={styles.content}>
              {item.component({ ...props, onClose }, tabRef)}
            </div>
          </Tabs.TabPane>
        ))}
      </Tabs>
    </Modal>
  );
});
