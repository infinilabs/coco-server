import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import { Modal, Tabs } from 'antd';

import Version from './Version';
import Code from './Code';
import EULA from './EULA';
import styles from './index.module.less';

export const DATE_FORMAT = 'YYYY.MM.DD HH:mm';

const LicenseModal = forwardRef((props, ref) => {
  LicenseModal.displayName = 'LicenseModal';
  const [visible, setVisible] = useState(false);
  const tabRef = useRef(null);
  const { t } = useTranslation();

  const tabs = [
    {
      key: 'version',
      title: t('license.titles.version'),
      component: Version
    },
    {
      key: 'license',
      title: t('license.titles.license'),
      component: Code
    },
    {
      key: 'eula',
      title: t('license.titles.eula'),
      component: EULA
    }
  ];

  const onOpen = () => {
    setVisible(true);
  };

  const onClose = () => {
    tabRef.current?.resetCode?.();
    setVisible();
  };

  useImperativeHandle(ref, () => ({
    open: onOpen,
    close: onClose
  }));

  return (
    <Modal
      closable
      destroyOnClose
      footer={null}
      open={visible}
      width={560}
      wrapClassName={styles.systemLicense}
      onCancel={onClose}
    >
      <Tabs
        defaultActiveKey='version'
        items={tabs.map(item => ({
          key: item.key,
          label: <div className='px-12px'>{item.title}</div>,
          children: <div className={styles.content}>{item.component({ ...props, onClose }, tabRef)}</div>
        }))}
      />
    </Modal>
  );
});

export default LicenseModal;
