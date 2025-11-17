import { createUser } from '@/service/api/security';

import { EditForm } from '../modules/EditForm';
import { Modal } from 'antd';

import copy from 'copy-to-clipboard';

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();

  const onSubmit = async (params: any, before?: () => void, after?: () => void) => {
    if (before) before();
    const res = await createUser(params);
    if (res?.data?.password) {
      Modal.success({
        title: t('common.addSuccess'),
        width: 530,
        content: (
          <div className='mt-12px'>
            <div className='mb-12px break-all'>
              {t('page.user.new.copyPassword')}
            </div>
            <div className="rounded bg-gray-100 py-[3px] pl-1em text-gray-500 leading-[1.4em]">
              {res?.data?.password}
            </div>
          </div>
        ),
        cancelButtonProps: { style: { display: 'none'} },
        okText: t('common.copy'),
        onOk: () => {
          const isCopy = copy(res?.data?.password);
          if (isCopy) {
            window.$message?.success(t('common.copySuccess'));
            nav(`/security?tab=user`);
          }
        }
      });
    }
    if (after) after();
  };

  return (
    <div className='h-full min-h-500px'>
      <ACard
        bordered={false}
        className='min-h-full flex-col-stretch sm:flex-1-hidden card-wrapper'
      >
        <div className='mb-30px ml--16px flex items-center text-lg font-bold'>
          <div className='mr-20px h-1.2em w-10px bg-[#1677FF]' />
          <div>{t(`page.user.new.title`)}</div>
        </div>
        <div className='px-30px'>
          <EditForm
            actionText={t('common.save')}
            onSubmit={onSubmit}
          />
        </div>
      </ACard>
    </div>
  );
}
