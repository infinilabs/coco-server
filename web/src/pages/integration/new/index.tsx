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
    <div className="bg-white pt-15px pb-15px min-h-full">
      <div className='mb-4 flex items-center text-lg font-bold'>
        <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
        <div>{t(`page.integration.form.title.new`)}</div>
      </div>
      <div className="px-30px">
        <EditForm onSubmit={onSubmit} actionText={t('common.save')}/>
      </div>
    </div>
  )
}