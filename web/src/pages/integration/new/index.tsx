import { createIntegration } from '@/service/api/integration';
import { EditForm } from '../modules/EditForm';

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  
  const onSubmit = async (params, before, after) => {
    if (before) before()
    const res = await createIntegration({
      ...params,
      enabled: true,
    })
    if (res?.data?.result === 'created') {
      window.$message?.success(t('common.addSuccess'));
      res?.data?._id && nav(`/integration/edit/${res.data._id}`)
    }
    if (after) after()
  }

  return (
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        className="min-h-full flex-col-stretch sm:flex-1-hidden card-wrapper"
      >
        <div className='ml--16px mb-4 flex items-center text-lg font-bold'>
          <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
          <div>{t(`page.integration.form.title.new`)}</div>
        </div>
        <div className="px-30px">
          <EditForm onSubmit={onSubmit} actionText={t('common.save')}/>
        </div>
      </ACard>
    </div>
  )
}