import { getSiderCollapse } from '@/store/slice/app';

import ButtonIcon from '../stateless/custom/ButtonIcon';
import { SketchOutlined } from '@ant-design/icons';
import { fetchApplicationInfo } from '@/service/api';
import License from '.';
import { useMemo, useRef } from 'react';
import { Spin } from 'antd';

const LicenseTrigger = memo(({ className }) => {
  const { t } = useTranslation();

  const siderCollapse = useAppSelector(getSiderCollapse);
  const licenseRef = useRef(null)

  const { data: res, loading } = useRequest(fetchApplicationInfo, {
    manual: false,
  });

  const { data } = res || {}

  const text = useMemo(() => {
    if (data?.application?.version?.number) {
      return `${t('icon.about')} (${data?.application?.version?.number})`
    }
    return `${t('icon.about')}`
  }, [data])

  return (
    <>
      <ButtonIcon
        className={className}
        tooltipContent={null}
        tooltipPlacement="right"
        onClick={() => {
          if (!loading) {
            licenseRef.current?.open()
          }
        }}
        justify={'left'}
      >
        <Spin spinning={loading} size="small">
          <div className="flex gap-8px">
            <SketchOutlined className="text-14px"/>
            { !siderCollapse && <span className="text-14px">{text}</span> }
          </div>
        </Spin>
      </ButtonIcon>
      <License
        ref={licenseRef}
        loading={loading}
        application={data?.application}
      />
    </>
  );
});

export default LicenseTrigger;
