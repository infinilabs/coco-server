import Vditor from 'vditor';
import 'vditor/dist/index.css';
import { Select } from 'antd';
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
  const onChangeRef = useRef(onChange);

  useEffect(() => {
    onChangeRef.current = onChange;
  });

  useEffect(() => {
    if (!containerRef.current || vdRef.current) return;
    const vd = new Vditor(containerRef.current, {
      value: value ?? '',
      height,
      mode: 'wysiwyg',
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
        // 粘贴/选择图片 → /attachment/_upload → 以 markdown 图片插回光标处
        handler(files: File[]): Promise<string> {
          return uploadAttachment(Array.from(files))
            .then((res: any) => {
              const ids: string[] = res?.attachments || [];
              if (!ids.length) throw new Error('upload failed');
              const succMap: Record<string, string> = {};
              files.forEach((f, i) => {
                const id = ids[i];
                if (id) succMap[f.name] = `/attachment/${id}`;
              });
              // vditor 约定的响应格式:自动在光标处插入 succMap 里的 markdown
              return JSON.stringify({
                msg: '',
                code: 0,
                data: { errFiles: [], succMap }
              });
            })
            .catch((err: unknown) =>
              JSON.stringify({ msg: String(err).slice(0, 80), code: -1, data: {} })
            );
        }
      },
      after() {
        readyRef.current = true;
        if (value) vd.setValue(value);
      },
      input: (v: string) => onChangeRef.current?.(v)
    });
    vdRef.current = vd;

    const onResize = () => {
      if (containerRef.current && vdRef.current) {
        // vditor 自适应容器宽度;高度固定
        vdRef.current.vditor.element.style.width = '100%';
      }
    };
    window.addEventListener('resize', onResize);
    return () => {
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
