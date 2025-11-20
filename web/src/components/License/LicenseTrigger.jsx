import { useRef } from 'react';
import { Spin } from 'antd';
import { SketchOutlined } from '@ant-design/icons';

import { getSiderCollapse } from '@/store/slice/app';
import ButtonIcon from '../stateless/custom/ButtonIcon';
import License from '.';
import { fetchApplicationInfo } from '@/service/api';
import { useRequest } from '@sa/hooks';

const LicenseTrigger = memo(({ className }) => {
  const { t } = useTranslation();

  const siderCollapse = useAppSelector(getSiderCollapse);

  const { data, loading } = useRequest(fetchApplicationInfo, {
    manual: false
  });

  const text = data?.application?.version?.number
    ? `${t('icon.about')} (${data?.application?.version?.number})`
    : `${t('icon.about')}`;

  //
  const licenseRef = useRef(null);

  return (
    <>
      <ButtonIcon
        className={className}
        justify='left'
        tooltipContent={null}
        tooltipPlacement='right'
        onClick={() => {
          if (!loading) {
            licenseRef.current?.open();
          }
        }}
      >
        <Spin
          size='small'
          spinning={loading}
        >
          <div className='flex gap-8px'>
            <SketchOutlined className='text-14px' />
            {!siderCollapse && <span className='text-14px'>{text}</span>}
          </div>
        </Spin>
      </ButtonIcon>
      <License
        application={data?.application}
        loading={loading}
        ref={licenseRef}
      />
    </>
  );
});

export default LicenseTrigger;
