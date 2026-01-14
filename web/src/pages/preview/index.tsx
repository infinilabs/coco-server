import { DocDetail } from '@infinilabs/doc-detail';
import { Button, Result, Spin, Typography } from 'antd';
import { useSearchParams } from 'react-router-dom';
import { filesize } from 'filesize';

import { request } from '@/service/request';
import logoLight from '@/assets/imgs/coco-logo-text-light.svg';
import logoDark from '@/assets/imgs/coco-logo-text-dark.svg';
import DateTime from '@/components/DateTime';

export function Component() {
  const [searchParams] = useSearchParams();
  const id = searchParams.get('document');

  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<any>();
  const [error, setError] = useState<any>();
  const [sourceUrl, setSourceUrl] = useState<string>();

  useMount(async () => {
    try {
      if (!id) {
        throw new Error('地址栏缺少必需的 document 参数。');
      }

      const { data } = await request({
        method: 'get',
        url: `/document/${id}`
      });

      const dataSource = data._source;

      const { data: ownerData } = await request({
        method: 'post',
        url: `/entity/card/user/${dataSource._system.owner_id}`
      });

      setData({ ...dataSource, owner: ownerData });
    } catch (error) {
      if (error instanceof Error) {
        setError(error.message);
      } else {
        setError(error);
      }
    } finally {
      setLoading(false);
    }
  });

  const renderContent = () => {
    if (sourceUrl) {
      return (
        <div className='mt-30 flex justify-center'>
          <div className='border border-border-secondary rounded-lg bg-black/3 px-6 py-10 dark:bg-white/7'>
            <div className='font-bold'>您将离开 Coco AI，打开文件的原始链接</div>

            <div className='mt-1'>Coco AI 无法保证外部链接的可用性与安全性</div>

            <div className='mt-4'>
              <Typography.Text type='secondary'>{sourceUrl}</Typography.Text>
            </div>

            <Button
              className='mt-10'
              shape='round'
              size='large'
              type='primary'
              onClick={() => {
                window.open(sourceUrl);
              }}
            >
              继续访问
            </Button>
          </div>
        </div>
      );
    }

    if (loading) {
      return (
        <Spin
          fullscreen
          percent='auto'
          spinning={loading}
        />
      );
    }

    if (error) {
      return (
        <div className='h-full flex flex-col justify-center'>
          <Result
            status='404'
            subTitle={String(error)}
            title='抱歉，出错了'
            extra={
              <Button
                type='primary'
                onClick={() => {
                  window.location.reload();
                }}
              >
                重新加载
              </Button>
            }
          />
        </div>
      );
    }

    return (
      <DocDetail
        data={{
          ...data,
          size: filesize(data?.size ?? 0),
          created: (
            <DateTime
              showTooltip={false}
              value={data?.created}
            />
          ),
          updated: (
            <DateTime
              showTooltip={false}
              value={data?.updated}
            />
          )
        }}
        openSourceButtonProps={{
          onClick() {
            setSourceUrl(data?.url);
          }
        }}
      />
    );
  };

  return (
    <div className='h-screen flex flex-col bg-container children:px-40'>
      <div className='h-20 flex items-center justify-between border-b border-border-secondary'>
        <div className='children:h-10'>
          <img
            className='dark:hidden'
            src={logoLight}
          />

          <img
            className='hidden dark:block'
            src={logoDark}
          />
        </div>

        <span className='text-xl font-bold'>文档预览</span>
      </div>

      <div className='mt-8 flex-1 overflow-hidden'>{renderContent()}</div>
    </div>
  );
}
