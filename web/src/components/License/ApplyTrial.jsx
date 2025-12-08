/* eslint-disable no-nested-ternary */
import { Alert, Button, Checkbox, Divider, Form, Spin, Typography } from 'antd';
import { ModalForm, ProFormText } from '@ant-design/pro-components';
import { CheckCircleFilled, CloseCircleFilled, InfoCircleTwoTone } from '@ant-design/icons';
import PropTypes from 'prop-types';

import LicenseInfo from './LicenseInfo';
import { getLocale } from '@/store/slice/app';
import { requestTrialLicense } from '@/service/api/license';
import './ApplyTrial.scss';

ApplyTrial.propTypes = {
  loading: PropTypes.bool,
  application: PropTypes.object,
  license: PropTypes.object,
  onLicenseApply: PropTypes.func.isRequired
};

export default function ApplyTrial(props) {
  const { loading = false, application = {}, license, onLicenseApply } = props;

  const { t } = useTranslation();
  const { defaultRequiredRule, formRules } = useFormRules();
  const [form] = Form.useForm();
  const locale = useAppSelector(getLocale);
  const [open, setOpen] = useState(false);
  const defaultState = {
    status: -1,
    error_msg: '',
    licensed: false
  };
  const [state, setState] = useState(defaultState);

  const customProps = state.status !== -1 ? { submitter: false } : {};

  return (
    <>
      {(!license || license.license_type === 'UnLicensed') && (
        <Button
          className='!pl-0'
          type='link'
          onClick={() => setOpen(true)}
        >
          {t('license.actions.trial')}
        </Button>
      )}
      <ModalForm
        autoFocusFirstInput
        form={form}
        open={open}
        title={t('license.actions.trial')}
        modalProps={{
          destroyOnClose: true,
          bodyProps: {
            style: {
              paddingTop: 8
            }
          },
          onCancel: () => {
            setOpen(false);
            setState({ ...state, ...defaultState });
          },
          okText: t('license.actions.submit'),
          footer: () => null
        }}
        {...customProps}
        width={560}
        onFinish={async values => {
          const res = await requestTrialLicense({
            ...values,
            locale,
            product: 'coco',
            version: application?.version?.number
          });
          if (res?.data?.acknowledged) {
            const ack = { status: 1 };
            if (res?.data?.license) {
              ack.licensed = true;
              onLicenseApply(res?.data?.license);
            }
            setState({ ...state, ...ack });
          } else {
            let error_msg = '';
            if (typeof res === 'undefined') {
              error_msg = t('license.tips.timeout.organization');
            } else if (res?.error?.reason) {
              error_msg = res.error.reason;
            }
            setState({ ...state, status: 0, error_msg });
          }
          return false;
        }}
      >
        {state.status === -1 ? (
          <>
            <Form.Item>
              <div className='break-all'>
                <InfoCircleTwoTone className='mr-6px' />
                <span>{t('license.tips.trial')}</span>
              </div>
            </Form.Item>
            <ProFormText
              fieldProps={{ maxLength: 100 }}
              label={t('license.labels.organization')}
              name='organization'
              rules={[defaultRequiredRule]}
            />
            <ProFormText
              fieldProps={{ maxLength: 50 }}
              label={t('license.labels.contact')}
              name='contact'
              rules={[defaultRequiredRule]}
            />
            <ProFormText
              fieldProps={{ maxLength: 50 }}
              label={t('license.labels.email')}
              name='email'
              rules={formRules.email}
            />
            <ProFormText
              fieldProps={{ maxLength: 20 }}
              label={t('license.labels.phone')}
              name='phone'
              rules={[defaultRequiredRule]}
            />
            <Form.Item
              name='agreement'
              rules={[defaultRequiredRule]}
              valuePropName='checked'
            >
              <Checkbox>
                {t('license.labels.agree')}{' '}
                <Typography.Link
                  href='https://infinilabs.cn/agreement/easysearch/'
                  target='_blank'
                >
                  {t('license.labels.agreement')}
                </Typography.Link>
              </Checkbox>
            </Form.Item>
          </>
        ) : null}
        {state.status !== -1 && (
          <div className='min-h-360px flex flex-col items-center justify-center gap-16px'>
            {state.status === 0 ? (
              <>
                <CloseCircleFilled className='mb-10px mt-0 text-48px text-[#ff0000]' />
                {state.error_msg ? (
                  <Alert
                    description={state.error_msg}
                    message={t('license.tips.error')}
                    type='error'
                  />
                ) : null}

                <div>{t('license.tips.failed')}</div>
                <div>
                  {t('license.tips.website')}
                  <Typography.Link
                    href='https://infinilabs.cn/company/contact'
                    target='_blank'
                  >
                    https://infinilabs.cn/company/contact
                  </Typography.Link>
                </div>
              </>
            ) : null}
            {state.status === 1 ? (
              <>
                <CheckCircleFilled className='mb-10px mt-0 text-48px text-[#836efe]' />
                <div className='break-all'>{t('license.tips.succeeded')}</div>
                <div className='min-h-'>
                  {loading ? (
                    <Spin size='small' />
                  ) : state.licensed && license?.license_type ? (
                    <>
                      <Divider orientation='left'>{t('license.titles.license')}</Divider>
                      <div>
                        <LicenseInfo license={license} />
                      </div>
                    </>
                  ) : null}
                </div>
              </>
            ) : null}
          </div>
        )}
      </ModalForm>
    </>
  );
}
