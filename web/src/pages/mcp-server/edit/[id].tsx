import { type LoaderFunctionArgs, useLoaderData } from "react-router-dom";
import { useRequest } from '@sa/hooks';
import { getMCPServer, updateMCPServer } from '@/service/api/mcp-server';
import { EditForm } from '../modules/EditForm';

export function Component() {
  const id = useLoaderData();
  const { t } = useTranslation();

  const { data, run, loading } = useRequest(getMCPServer, {
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
    const res = await updateMCPServer(data._source.id, { id: data._source.id, ...params})
    if (res.data?.result === 'updated') {
      window.$message?.success(t('common.updateSuccess'));
    }
    if (after) after()
  }

  useEffect(() => {
    run(id)
  }, [])

  return (
    <div className="bg-white pt-15px pb-15px min-h-full">
      <div className='mb-4 flex items-center text-lg font-bold'>
        <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
        <div>{t(`route.ai-assistant_edit`)}</div>
      </div>
      <EditForm initialValues={data?._source || {}} loading={loading} onSubmit={onSubmit} mode="edit" />
    </div>
  )
}

export async function loader({ params, ...rest }: LoaderFunctionArgs) {
  return params.id;
}