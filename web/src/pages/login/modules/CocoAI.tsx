import { Button, Spin } from 'antd';
import Clipboard from 'clipboard';

import { fetchAccessToken } from '@/service/api';
import { useRequest } from '@sa/hooks';

const CocoAI = ({ provider, requestID }: { readonly provider: string | null; readonly requestID: string | null }) => {
  const { t } = useTranslation();

  const linkRef = useRef<HTMLButtonElement>(null);
  const copyRef = useRef<HTMLButtonElement>(null);
  const router = useRouterPush();
  const clickRef = useRef(false)

  const { data, loading, run } = useRequest(fetchAccessToken, {
      manual: true,
  });

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
    run()
  }, [])

  const url = useMemo(() => {
    if (data?.access_token && requestID && provider) {
      return `coco://oauth_callback?code=${data?.access_token}&request_id=${requestID}&provider=${provider}expire_in=${data?.expire_in}`
    }
    return ''
  }, [data, requestID, provider])

  useEffect(() => {
    if (url) {
      setTimeout(() => {
        if (!clickRef.current) {
          clickRef.current = true
          linkRef.current?.click();
        }
      }, 5000);
    }
  }, [url]);

  useEffect(() => {
    if (copyRef.current && url) {
      initClipboard(url);
    }
  }, [url, copyRef.current]);

  return (
    <>
      <Spin spinning={loading}>
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
          onClick={() => {
            clickRef.current = true
          }}
        >
          {t('page.login.cocoAI.launchCocoAI')}
        </Button>
        <div
          className="m-b-12px text-16px color-[var(--ant-color-text)]"
          style={{ wordBreak: 'break-all' }}
        >
          {t('page.login.cocoAI.copyDesc')}
        </div>
        <div className="relative m-b-16px">
          <pre
            className="relative whitespace-pre-wrap border border-gray-300 rounded-4px bg-gray-100 p-8px"
            style={{ wordBreak: 'break-all' }}
          >
            <code className="text-gray-700">{url}</code>
          </pre>
          <Button
            className="absolute right-0 bottom-0 z-1 p-4px"
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
      </Spin>
    </>
  );
};

export default CocoAI;
