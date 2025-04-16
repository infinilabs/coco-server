import { Button, Divider } from 'antd';
import Clipboard from 'clipboard';

import { Preview } from './Preview';

export const InsertCode = memo(props => {
  const { id, token } = props;

  const { t } = useTranslation();
  const copyRef = useRef<HTMLButtonElement>(null);

  const initClipboard = (text?: string) => {
    if (!copyRef.current || !text) return;

    const clipboard = new Clipboard(copyRef.current, {
      text: () => text
    });

    clipboard.on('success', () => {
      window.$message?.success(t('common.copySuccess'));
    });
  };

  const code = useMemo(() => {
    return `<div id="searchbox" style="margin: 10px 0;"></div>
<script type="module" >
    import { searchbox } from "${window.location.origin}/integration/${id}/widget";
    searchbox({container: "#searchbox"});
</script>`;
  }, [id, token]);

  useEffect(() => {
    if (copyRef.current) {
      initClipboard(code);
    }
  }, [code, copyRef.current]);

  const borderColor = 'var(--ant-color-border)';

  return (
    <div
      className="relative max-w-860px w-[100%] border border-[#d9d9d9] rounded-[var(--ant-border-radius)] px-24px py-30px"
      style={{ borderColor }}
    >
      <div className="mb-12px text-lg font-bold">{t('page.integration.code.title')}</div>
      <div className="color-var(--ant-color-text) mb-12px">{t('page.integration.code.desc')}</div>
      <pre
        className="color-var(--ant-color-text) relative rounded-[var(--ant-border-radius)] bg-[var(--ant-color-border)] p-12px"
        style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}
      >
        {code}
        <div className="absolute right-0 top-0 z-1 flex items-center p-4px">
          <Button
            className="p-0"
            ref={copyRef}
            title={t('common.copy')}
            type="link"
          >
            <SvgIcon
              className="text-16px"
              icon="mdi:content-copy"
            />
          </Button>
        </div>
      </pre>
      <div className="text-right">
        <Preview params={{ id, server: `${window.location.origin}`, token }}>
          <Button
            className="mt-12px"
            size="large"
            type="primary"
          >
            <SvgIcon
              className="text-18px"
              icon="mdi:web"
            />{' '}
            {t('page.integration.code.preview')}
          </Button>
        </Preview>
      </div>
    </div>
  );
});
