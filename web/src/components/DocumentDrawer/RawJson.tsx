import { Button, Spin } from 'antd';
import { CopyOutlined } from '@ant-design/icons';

/**
 * 原始 JSON 视图：pretty-print 的完整文档响应（ES found/_id/_source 全量），
 * 带一键复制。纯展示组件——取数由调用方负责（详情页已有全量响应，抽屉懒加载）。
 */
export default function RawJson({ value, loading }: { value?: unknown; loading?: boolean }) {
    const { t } = useTranslation();
    if (loading) return <Spin className="my-24px" />;
    const text = value == null ? '' : JSON.stringify(value, null, 2);
    return (
        <div className="relative">
            <Button
                className="absolute right-8px top-8px z-1"
                icon={<CopyOutlined />}
                onClick={() => navigator.clipboard?.writeText(text)}
                size="small"
            >
                {t('common.copy')}
            </Button>
            <pre className="m-0 p-12px rounded-8px bg-#F6F8FA dark:bg-[rgb(var(--layout-bg-color))] text-12px leading-20px overflow-auto max-h-[calc(100vh-260px)] whitespace-pre-wrap break-all">
                {text || '{}'}
            </pre>
        </div>
    );
}
