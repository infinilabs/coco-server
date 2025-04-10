import { useRequest } from '@sa/hooks';
import { Col, Row } from 'antd';
import { type LoaderFunctionArgs, useLoaderData } from 'react-router-dom';

import { fetchIntegration, updateIntegration } from '@/service/api/integration';

import { EditForm } from '../modules/EditForm';
import { InsertCode } from '../modules/InsertCode';

export function Component() {
  const id = useLoaderData();
  const { t } = useTranslation();

  const { data, loading, run } = useRequest(fetchIntegration, {
    manual: true
  });

  const onSubmit = async (params, before, after) => {
    if (!data?._source?.id) return;
    if (before) before();
    const res = await updateIntegration({ id: data._source.id, ...params });
    if (res.data?.result === 'updated') {
      window.$message?.success(t('common.updateSuccess'));
    }
    if (after) after();
  };

  useEffect(() => {
    run(id);
  }, []);

  return (
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        className="min-h-full flex-col-stretch sm:flex-1-hidden card-wrapper"
      >
        <div className="mb-4 ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          <div>{t(`page.integration.form.title.edit`)}</div>
        </div>
        <Row
          className="px-30px"
          gutter={24}
        >
          <Col>
            <EditForm
              actionText={t('common.update')}
              loading={loading}
              record={data?._source}
              onSubmit={onSubmit}
            />
          </Col>
          <Col flex="1">
            <InsertCode
              id={data?._source?.id}
              token={data?._source?.token}
            />
          </Col>
        </Row>
      </ACard>
    </div>
  );
}

export async function loader({ params, ...rest }: LoaderFunctionArgs) {
  return params.id;
}
