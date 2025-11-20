import { EditOutlined } from '@ant-design/icons';
import { Button, Input, Spin } from 'antd';
import { useEffect, useState } from 'react';
import PropTypes from 'prop-types';

import { applyLicense, fetchLicense } from '@/service/api/license';
import ApplyTrial from './ApplyTrial';
import LicenseInfo from './LicenseInfo';

import styles from './Code.module.less';

Code.propTypes = {
  application: PropTypes.object.isRequired
};

export default function Code(props, ref) {
  const { application } = props;

  const { t } = useTranslation();

  const [isEdit, setIsEdit] = useState(false);
  const [license, setLicense] = useState();
  const [code, setCode] = useState();
  const [loading, setLoading] = useState(false);

  const textareaRef = useRef(null);
  const buttonRef = useRef(null);

  const fetchData = async () => {
    const res = await fetchLicense();
    if (res?.data) {
      setLicense(res.data);
    } else {
      setLicense();
    }
  };

  const onUpdate = async code => {
    if (code) {
      setLoading(true);
      const res = await applyLicense(code);
      if (res?.data?.acknowledged) {
        window?.$message?.success(t('common.operateSuccess'));
        setIsEdit(false);
        setCode();
        fetchData();
      }
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  useEffect(() => {
    const handleClickOutside = event => {
      if (isEdit) {
        const isTextarea = textareaRef.current?.resizableTextArea?.textArea?.contains(event.target);
        const isButton = buttonRef.current?.contains(event.target);
        if (!isTextarea && !isButton) {
          setIsEdit(false);
        }
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isEdit]);

  useImperativeHandle(ref, () => ({
    resetCode: () => {
      setIsEdit(false);
      setCode();
    }
  }));

  return (
    <Spin spinning={loading}>
      <div className={styles.license}>
        <div className={styles.header}>
          <LicenseInfo license={license} />
        </div>
        <div className={styles.licenseBox}>
          {isEdit ? (
            <Input.TextArea
              autoFocus
              ref={textareaRef}
              rows={5}
              value={code}
              onChange={e => setCode(e.target.value)}
            />
          ) : (
            <div className={styles.edit}>
              <a onClick={() => setIsEdit(true)}>
                <EditOutlined />
              </a>
            </div>
          )}
        </div>
        <div className={styles.footer}>
          <div className='h-28px flex items-center'>
            <ApplyTrial
              application={application}
              license={license}
              loading={loading}
              onLicenseApply={onUpdate}
            />
            <Button
              className='!pl-0'
              type='link'
              onClick={() => {
                window.open('https://infinilabs.cn/company/contact/');
              }}
            >
              {t('license.actions.buy')}
            </Button>
          </div>
          <Button
            ref={buttonRef}
            size='small'
            type='primary'
            onClick={() => onUpdate(code)}
          >
            {t('license.actions.apply')}
          </Button>
        </div>
      </div>
    </Spin>
  );
}
