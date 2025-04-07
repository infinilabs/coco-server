import { EditForm } from '../modules/EditForm';
import { type LoaderFunctionArgs, useLoaderData } from "react-router-dom";
import { useRequest } from '@sa/hooks';
import { fetchIntegration, updateIntegration } from '@/service/api/integration';
import { Col, Row } from 'antd';
import { InsertCode } from '../modules/InsertCode';

export function Component() {
  const id = useLoaderData();
  const { t } = useTranslation();

  const { data, run, loading } = useRequest(fetchIntegration, {
    manual: true,
  });

  const onSubmit = async (params, before, after) => {
    if (!data?._source?.id) return;
    if (before) before()
    const res = await updateIntegration({ id: data._source.id, ...params})
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
        <div className='ml--16px mb-4 flex items-center text-lg font-bold'>
        <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
        <div>{t(`page.integration.form.title.edit`)}</div>
      </div>
        <Row className="px-30px" gutter={24}>
          <Col>
            <EditForm record={data?._source} loading={loading} onSubmit={onSubmit} actionText={t('common.update')} />
          </Col>
          <Col flex="1">
            <InsertCode token={data?._source?.token} id={data?._source?.id}/>
          </Col>
        </Row>
      </ACard>
    </div>
  )
}

export async function loader({ params, ...rest }: LoaderFunctionArgs) {
  return params.id;
}