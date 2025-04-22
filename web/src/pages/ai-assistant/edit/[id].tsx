import { type LoaderFunctionArgs, useLoaderData } from "react-router-dom";
import { useRequest } from '@sa/hooks';
import { getAssistant, updateAssistant } from '@/service/api/assistant';
import { EditForm } from '../modules/EditForm';

export function Component() {
  const id = useLoaderData();
  const { t } = useTranslation();

  const { data, run, loading } = useRequest(getAssistant, {
    manual: true,
  });

  const onSubmit = async (values, before, after) => {
    const params = {
      ...values,
      datasource: {
        ...(values.datasource || {}),
        ids: values.datasource?.ids?.includes('*') ? ['*'] : values.datasource?.ids,
      }
    };
    if (!data?._source?.id) return;
    if (before) before()
    const res = await updateAssistant(data._source.id, { id: data._source.id, ...params})
    if (res.data?.result === 'updated') {
      window.$message?.success(t('common.updateSuccess'));
    }
    if (after) after()
  }

  useEffect(() => {
    run(id)
  }, [])

  return (
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        className="min-h-full flex-col-stretch sm:flex-1-hidden card-wrapper"
      >
        <div className="mb-30px ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          <div>{t(`route.ai-assistant_edit`)}</div>
        </div>
        <div className="px-30px">
          <EditForm initialValues={data?._source || {}} loading={loading} onSubmit={onSubmit} mode="edit" />
        </div>
      </ACard>
    </div>
  )
}

export async function loader({ params, ...rest }: LoaderFunctionArgs) {
  return params.id;
}