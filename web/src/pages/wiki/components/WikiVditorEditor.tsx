import Vditor from 'vditor';
import 'vditor/dist/index.css';
import { Button, Select } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { uploadAttachment } from '@/service/api/ai-search';

export interface WikiEntityOption {
  value: string;
  type: string;
}

interface Props {
  value?: string;
  onChange?: (content: string) => void;
  darkMode?: boolean;
  entityOptions?: WikiEntityOption[];
  height?: number;
}

/**
 * Wiki 富文本编辑器 (Vditor WYSIWYG):所见即所得 + 即时渲染 + 分屏源码三种
 * 模式;粘贴/选择图片自动走附件上传并插入;[[type:name]] wikilink 以纯文本
 * 形式编辑,阅读与预览视图渲染为可点击胶囊。实体选择器直接在光标处插入
 * 解析过的 wikilink。
 */
export function WikiVditorEditor({
  value,
  onChange,
  darkMode,
  entityOptions = [],
  height = 520
}: Props) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const vdRef = useRef<Vditor | null>(null);
  const readyRef = useRef(false);
  const pickImageRef = useRef<(() => void) | null>(null);
  const onChangeRef = useRef(onChange);
  // 选中图片后浮出的宽度工具条
  const [imgTool, setImgTool] = useState<{ x: number; y: number; img: HTMLImageElement } | null>(null);

  // 尺寸工具条:按百分比设定宽度,写入 alt(名称|像素)持久化
  const applyImgWidth = (pct: number) => {
    if (!imgTool) return;
    const img = imgTool.img;
    const px = Math.round((containerRef.current?.clientWidth || 800) * pct);
    img.style.width = px + 'px';
    img.style.height = 'auto';
    const base = (img.alt || '图片').split('|')[0];
    img.alt = `${base}|${px}`;
    setImgTool(null);
  };

  useEffect(() => {
    onChangeRef.current = onChange;
  });

  useEffect(() => {
    if (!containerRef.current || vdRef.current) return;
    const vd = new Vditor(containerRef.current, {
      value: value ?? '',
      height,
      mode: 'wysiwyg',
      lang: 'zh_CN',
      theme: darkMode ? 'dark' : 'classic',
      cache: { enable: false },
      counter: { enable: true },
      preview: { theme: { current: darkMode ? 'dark' : 'light' } },
      toolbar: [
        'headings',
        'bold',
        'italic',
        'strike',
        '|',
        'list',
        'ordered-list',
        'quote',
        '|',
        'code',
        'inline-code',
        'table',
        'link',
        { name: 'insert-image', tipPosition: 'n', tip: '上传图片', className: 'wiki-vditor-image', icon: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="M21 15l-5-5L5 21"/></svg>', click: () => pickImageRef.current?.() },
        '|',
        'undo',
        'redo',
        '|',
        'edit-mode',
        'export',
        'preview'
      ],
      toolbarConfig: { pin: true },
      upload: {
        accept: 'image/*',
        // 粘贴/选择图片 → /attachment/_upload → 成功后自行 insertValue 插入
        // markdown(vditor 契约:handler 的返回值会显示为提示气泡而非插入)
        handler(files: File[]): Promise<string> {
          return uploadAttachment(Array.from(files))
            .then((res: any) => {
              const ids: string[] = res?.data?.attachments || res?.attachments || [];
              if (!ids.length) throw new Error('upload failed: no attachment ids');
              const md = files
                .map((f, i) => {
                  const id = ids[i];
                  if (!id) return '';
                  const name = f.name.replace(/\.[^.]+$/, '') || `image-${i + 1}`;
                  return `![${name}](/attachment/${id})`;
                })
                .filter(Boolean)
                .join('\n');
              if (md) vd.insertValue(md);
              return '';
            })
            .catch((err: unknown) => {
              window.$message?.error(String(err).slice(0, 80));
              return '';
            });
        }
      },
      after() {
        readyRef.current = true;
        if (value) vd.setValue(value);
      },
      input: (v: string) => onChangeRef.current?.(v)
    });
    vdRef.current = vd;

    // 点击 wysiwyg 里的图片 → 浮出尺寸工具条
    const onEditorClick = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      if (target.tagName !== 'IMG') {
        setImgTool(null);
        return;
      }
      const b = target.getBoundingClientRect();
      setImgTool({ x: b.left + window.scrollX, y: b.top + window.scrollY - 44, img: target as HTMLImageElement });
    };
    containerRef.current.addEventListener('click', onEditorClick);

    // 隐藏文件选择器:工具栏图片按钮/编辑器内图片上传共用
    const fileInput = document.createElement('input');
    fileInput.type = 'file';
    fileInput.accept = 'image/*';
    fileInput.multiple = true;
    fileInput.style.display = 'none';
    fileInput.onchange = () => {
      const files = Array.from(fileInput.files || []);
      if (!files.length) return;
      uploadAttachment(files)
        .then((res: any) => {
          const ids: string[] = res?.data?.attachments || res?.attachments || [];
          ids.forEach((id, i) => {
            const name = files[i]?.name?.replace(/\.[^.]+$/, '') || `image-${i + 1}`;
            vd.insertValue(`\n![${name}](/attachment/${id})\n`);
          });
        })
        .catch((err: unknown) => console.error('image upload failed', err))
        .finally(() => {
          fileInput.value = '';
        });
    };
    containerRef.current.parentElement?.appendChild(fileInput);
    pickImageRef.current = () => fileInput.click();

    const onResize = () => {
      if (containerRef.current && vdRef.current) {
        // vditor 自适应容器宽度;高度固定
        vdRef.current.vditor.element.style.width = '100%';
      }
    };
    window.addEventListener('resize', onResize);
    return () => {
      containerRef.current?.removeEventListener('click', onEditorClick);
      window.removeEventListener('resize', onResize);
      readyRef.current = false;
      try {
        vdRef.current?.destroy();
      } catch {
        /* already torn down */
      }
      vdRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [darkMode, height]);

  // 外部内容重置(打开编辑器载入文章)时同步 — 仅在编辑器就绪后
  useEffect(() => {
    if (!readyRef.current || !vdRef.current || value == null) return;
    try {
      const current = vdRef.current.getValue();
      if (current !== value) vdRef.current.setValue(value);
    } catch {
      /* editor not initialized yet */
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value]);

  const insertEntity = (name: string, type: string) => {
    vdRef.current?.insertValue(`[[${type}:${name}]] `);
  };

  return (
    <div className='w-full'>
      {imgTool && (
        <div
          className='fixed z-50 flex items-center gap-4px rounded-4px border border-solid border-[var(--ant-color-border-secondary)] bg-[var(--ant-color-bg-container)] px-6px py-4px shadow-md'
          style={{ left: imgTool.x, top: Math.max(imgTool.y, 60) }}
        >
          {[0.25, 0.5, 0.75, 1].map(p => (
            <Button key={p} onClick={() => applyImgWidth(p)} size='small'>
              {p * 100}%
            </Button>
          ))}
          <Button onClick={() => setImgTool(null)} size='small' type='text'>
            ✕
          </Button>
        </div>
      )}
      <div className='mb-4px flex items-center gap-8px'>
        <Select
          showSearch
          size='small'
          style={{ minWidth: 220 }}
          placeholder='插入实体链接(搜索实体名)'
          filterOption={(input, option) =>
            String(option?.value ?? '').toLowerCase().includes(input.toLowerCase())
          }
          value={null}
          onChange={(v: string) => {
            const opt = entityOptions.find(o => o.value === v);
            insertEntity(v, opt?.type || 'concept');
          }}
        >
          {entityOptions.map(o => (
            <Select.Option key={o.value} value={o.value}>
              {o.type ? `${o.type} · ${o.value}` : o.value}
            </Select.Option>
          ))}
        </Select>
      </div>
      <div ref={containerRef} className='w-full' />
    </div>
  );
}
