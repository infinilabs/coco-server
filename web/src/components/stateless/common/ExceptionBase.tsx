import { Button, Space, Typography } from 'antd';

type ExceptionType = '403' | '404' | '500';

interface Props {
  /**
   * Exception type
   *
   * - 403: no permission
   * - 404: not found
   * - 500: service error
   */
  type: ExceptionType;
}

const exceptionCopy: Record<ExceptionType, { titleKey: string; descKey: string }> = {
  '403': { titleKey: 'common.exception.title403', descKey: 'common.exception.desc403' },
  '404': { titleKey: 'common.exception.title404', descKey: 'common.exception.desc404' },
  '500': { titleKey: 'common.exception.title500', descKey: 'common.exception.desc500' }
};

/**
 * Full-page exception state (403/404/500). The old undraw illustration was
 * drawn for light backgrounds only and washed out in dark mode, so this is
 * pure CSS: a gradient status number over soft brand-color glows that adapt
 * to both themes, with a one-line cause and two ways out.
 */
const ExceptionBase: FC<Props> = memo(({ type }) => {
  const { t } = useTranslation();
  const nav = useNavigate();
  const copy = exceptionCopy[type];

  return (
    <div className='relative size-full min-h-520px flex-col-center overflow-hidden'>
      <div className='pointer-events-none absolute -top-100px -left-80px size-400px rounded-full bg-primary/10 blur-3xl dark:bg-primary/15' />
      <div className='pointer-events-none absolute -right-80px -bottom-120px size-420px rounded-full bg-info/10 blur-3xl dark:bg-info/15' />
      <div className='z-1 flex flex-col items-center gap-12px px-24px text-center'>
        <div className='bg-gradient-to-b from-primary-300 to-primary-700 bg-clip-text text-160px leading-none font-black text-transparent select-none dark:from-primary-200 dark:to-primary-600'>
          {type}
        </div>
        <Typography.Title level={3} className='!mb-0'>
          {t(copy.titleKey)}
        </Typography.Title>
        <Typography.Text type='secondary' className='max-w-480px'>
          {t(copy.descKey)}
        </Typography.Text>
        <Space className='mt-12px'>
          <Button onClick={() => nav(-1)}>{t('common.exception.backToPrev')}</Button>
          <Button
            type='primary'
            onClick={() => {
              nav('/');
            }}
          >
            {t('common.backToHome')}
          </Button>
        </Space>
      </div>
    </div>
  );
});

export default ExceptionBase;
