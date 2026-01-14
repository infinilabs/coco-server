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
  const { t } = useTranslation();

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
            <div className='font-bold'>{t('page.preview.hints.leave')}</div>

            <div className='mt-1'>{t('page.preview.hints.externalLinkWarning')}</div>

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
              {t('page.preview.buttons.continueVisiting')}
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
            title={t('page.preview.hints.failed')}
            extra={
              <Button
                type='primary'
                onClick={() => {
                  window.location.reload();
                }}
              >
                {t('page.preview.buttons.reload')}
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
        i18n={{
          labels: {
            type: t('page.preview.labels.type'),
            size: t('page.preview.labels.size'),
            createdBy: t('page.preview.labels.createdBy'),
            createdAt: t('page.preview.labels.createdAt'),
            updatedAt: t('page.preview.labels.updatedAt'),
            preview: t('page.preview.labels.preview'),
            aiInterpretation: t('page.preview.labels.aiInterpretation')
          },
          buttons: {
            openSource: t('page.preview.buttons.openSource')
          }
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

        <span className='text-xl font-bold'>{t('page.preview.title')}</span>
      </div>

      <div className='mt-8 flex-1 overflow-hidden'>{renderContent()}</div>
    </div>
  );
}
