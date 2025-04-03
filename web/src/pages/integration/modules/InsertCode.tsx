import { Button, Divider } from "antd";
import Clipboard from 'clipboard';
import { Preview } from "./Preview";

export const InsertCode = memo((props) => {
    
    const { id, token } = props;

    const { t } = useTranslation();
    const copyRef = useRef<HTMLButtonElement>(null)

    const initClipboard = (text?: string) => {
        if (!copyRef.current || !text) return;
    
        const clipboard = new Clipboard(copyRef.current, {
          text: () => text
        });
    
        clipboard.on('success', () => {
          window.$message?.success(t('common.copySuccess'));
        });
    }

    const code = useMemo(() => {
        return `<div id="searchbox"></div>
<script type="module" >
    import { searchbox } from "${window.location.origin}/integration/${id}/widget";
    searchbox({container: "#searchbox"});
</script>`
    }, [id, token])

    useEffect(() => {
        if (copyRef.current) {
          initClipboard(code)
        }
    }, [code, copyRef.current])

    const borderColor = 'var(--ant-color-border)'

    return (
        <div className="px-24px py-30px w-[100%] max-w-860px border border-[#d9d9d9] rounded-[var(--ant-border-radius)] relative" style={{ borderColor }}>
            <div className="text-lg font-bold mb-12px">{t('page.integration.code.title')}</div>
            <div className="mb-12px color-var(--ant-color-text)">{t('page.integration.code.desc')}</div>
            <pre className="bg-[var(--ant-color-border)] rounded-[var(--ant-border-radius)] color-var(--ant-color-text) p-12px relative" style={{ wordBreak: 'break-all', whiteSpace: 'pre-wrap'}}>
                {code}
                <div className="z-1 absolute right-0 top-0 p-4px flex items-center">
                    <Button title={t('common.copy')} ref={copyRef} type="link" className="p-0">
                        <SvgIcon className="text-16px" icon="mdi:content-copy" />
                    </Button>
                </div>
            </pre>
            <div className="text-right">
                <Preview params={{ id, token, server: `${window.location.origin}`}}>
                    <Button size="large" type="primary" className="mt-12px">
                        <SvgIcon className="text-18px" icon="mdi:web"/> {t('page.integration.code.preview')}
                    </Button>
                </Preview>
            </div>
        </div>
    )
})