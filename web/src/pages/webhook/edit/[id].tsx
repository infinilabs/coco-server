import { useRequest } from '@sa/hooks';
import { type LoaderFunctionArgs, useLoaderData } from 'react-router-dom';
import { fetchWebhook, updateWebhook } from '@/service/api/webhook';
import { WebhookForm } from '../modules/WebhookForm';

export function Component() {
  const id = useLoaderData();
  const { t } = useTranslation();

  const { data, loading, run } = useRequest(fetchWebhook, { manual: true });

  const onSubmit = async (params, before, after) => {
    if (!data?._source?.id) return;
    if (before) before();
    const res = await updateWebhook({ id: data._source.id, ...params });
    if (res.data?.result === 'updated') {
      window.$message?.success(t('common.updateSuccess'));
      run(data._source.id);
    }
    if (after) after();
  };

  useEffect(() => {
    run(id);
  }, []);

  return (
    <div className="h-full min-h-500px">
      <ACard bordered={false} className="min-h-full flex-col-stretch sm:flex-1-hidden card-wrapper">
        <div className="mb-30px ml--16px flex items中心 text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          <div>{t('page.webhook.form.title.edit')}</div>
        </div>
        <WebhookForm actionText={t('common.update')} loading={loading} record={data?._source} onSubmit={onSubmit} />
      </ACard>
    </div>
  );
}

export async function loader({ params }: LoaderFunctionArgs) {
  return params.id;
}