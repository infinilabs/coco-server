import { useRequest } from '@sa/hooks';
import { type LoaderFunctionArgs, useLoaderData } from 'react-router-dom';

import { EditForm } from '../modules/EditForm';
import { fetchUser, updateUser } from '@/service/api/security';

export function Component() {
  const id = useLoaderData();
  const { t } = useTranslation();
  const nav = useNavigate();

  const { data, loading, run } = useRequest(fetchUser, {
    manual: true
  });

  const onSubmit = async (params: any, before?: () => void, after?: () => void) => {
    if (!data?._source?.id) return;
    if (before) before();
    const req = {
      id: data._source.id,
      ...params,
    };
    const res = await updateUser(req);
    if (res.data?.result === 'updated') {
      window.$message?.success(t('common.updateSuccess'));
      nav(`/security?tab=user`);
    }
    if (after) after();
  };

  useEffect(() => {
    run(id as string);
  }, []);

  return (
    <div className='h-full min-h-500px'>
      <ACard
        bordered={false}
        className='min-h-full flex-col-stretch sm:flex-1-hidden card-wrapper'
      >
        <div className='mb-30px ml--16px flex items-center text-lg font-bold'>
          <div className='mr-20px h-1.2em w-10px bg-[#1677FF]' />
          <div>{t(`page.user.edit.title`)}</div>
        </div>
        <div className='px-30px'>
          <EditForm
            actionText={t('common.update')}
            loading={loading}
            record={data?._source}
            onSubmit={onSubmit}
          />
        </div>
      </ACard>
    </div>
  );
}

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export async function loader({ params, ...rest }: LoaderFunctionArgs) {
  return params.id;
}
