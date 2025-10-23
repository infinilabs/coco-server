import { createRole } from '@/service/api/role';

import { EditForm } from '../modules/EditForm';

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();

  const onSubmit = async (params: any, before?: () => void, after?: () => void) => {
    if (before) before();
    const data = {
      ...params,
      grants: {
        permissions: params.permission.feature
      }
    };
    const res = await createRole(data);
    if (res?.data?.result === 'created') {
      window.$message?.success(t('common.addSuccess'));
      nav(`/security?tab=role`);
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
          <div>{t(`page.role.new.title`)}</div>
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
