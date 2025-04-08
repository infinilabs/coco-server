import { Button } from 'antd';
import Clipboard from 'clipboard';

import { localStg } from '@/utils/storage';

const CocoAI = ({ provider, requestID }: { readonly provider: string | null; readonly requestID: string | null }) => {
  const { t } = useTranslation();

  const token = localStg.get('token');
  const url = token ? `coco://oauth_callback?code=${token}&request_id=${requestID}&provider=${provider}` : '';
  const linkRef = useRef<HTMLButtonElement>(null);
  const copyRef = useRef<HTMLButtonElement>(null);
  const router = useRouterPush();

  const initClipboard = (text?: string) => {
    if (!copyRef.current || !text) return;

    const clipboard = new Clipboard(copyRef.current, {
      text: () => text
    });

    clipboard.on('success', () => {
      window.$message?.success(t('common.copySuccess'));
    });
  };

  useEffect(() => {
    setTimeout(() => {
      linkRef.current?.click();
    }, 5000);
  }, []);

  useEffect(() => {
    if (copyRef.current) {
      initClipboard(url);
    }
  }, [url, copyRef.current]);

  return (
    <>
      <div
        className="m-b-12px text-16px color-[var(--ant-color-text)]"
        style={{ wordBreak: 'break-all' }}
      >
        {t('page.login.cocoAI.autoDesc')}
      </div>
      <Button
        className="m-b-16px p-0"
        href={url}
        ref={linkRef}
        type="link"
      >
        {t('page.login.cocoAI.launchCocoAI')}
      </Button>
      <div
        className="m-b-12px text-16px color-[var(--ant-color-text)]"
        style={{ wordBreak: 'break-all' }}
      >
        {t('page.login.cocoAI.copyDesc')}
      </div>
      <div className="group relative m-b-16px">
        <pre
          className="relative whitespace-pre-wrap border border-gray-300 rounded-4px bg-gray-100 p-8px"
          style={{ wordBreak: 'break-all' }}
        >
          <code className="text-gray-700">{url}</code>
        </pre>
        <Button
          className="absolute right-0 top-0 z-1 hidden p-4px group-hover:block"
          ref={copyRef}
          type="link"
        >
          <SvgIcon
            className="text-16px"
            icon="mdi:content-copy"
          />
        </Button>
      </div>
      <div
        className="m-b-12px text-16px"
        style={{ wordBreak: 'break-all' }}
      >
        {t('page.login.cocoAI.enterCocoServerDesc')}
      </div>
      <Button
        className="p-0"
        type="link"
        onClick={() => router.routerPushByKey('home')}
      >
        {t('page.login.cocoAI.enterCocoServer')}
      </Button>
    </>
  );
};

export default CocoAI;
