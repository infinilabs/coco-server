import { SaveOutlined } from '@ant-design/icons';
import { Button } from 'antd';
import { createContext, useContext } from 'react';
import { createPortal } from 'react-dom';
import { useTranslation } from 'react-i18next';

/**
 * Ontology-page parity: the save bar renders OUTSIDE the white Card, on the
 * page canvas below it (full-width footer look, see ontology/index.tsx).
 * Shells (ModuleShell / settings page) mount an empty slot element after
 * their Card and provide it via this context; SaveBar portals itself into
 * the slot when present and falls back to inline rendering otherwise.
 */
export const SaveBarSlotContext = createContext<HTMLElement | null>(null);

// 本体词表页的保存条样式（全站统一基准）：
// 左侧脏状态提示 + 右侧「放弃更改 / 保存」，未修改时两按钮禁用。
// sticky 贴布局滚动条底部；负 margin 抵消页面 16px 内边距直抵页缘。
const SaveBar = ({ dirty, loading, onDiscard, onSave }: { dirty: boolean; loading?: boolean; onDiscard: () => void; onSave: () => void }) => {
  const { t } = useTranslation();
  const slot = useContext(SaveBarSlotContext);

  const bar = (
    <div
      className='sticky bottom-0 z-20 -mx-16px mt-16px flex items-center gap-12px px-16px py-10px'
      style={{
        background: 'var(--ant-color-bg-container)',
        borderTop: '1px solid var(--ant-color-split)',
        marginBottom: '-16px'
      }}
    >
      <span style={{ color: dirty ? 'var(--ant-color-warning)' : 'var(--ant-color-text-quaternary)' }}>
        {dirty ? t('common.unsaved_changes') : t('common.all_saved')}
      </span>
      <div className='ml-auto flex gap-8px'>
        <Button disabled={!dirty} onClick={onDiscard}>
          {t('common.discard')}
        </Button>
        <Button disabled={!dirty} icon={<SaveOutlined />} loading={loading} onClick={onSave} type='primary'>
          {t('common.save')}
        </Button>
      </div>
    </div>
  );

  return slot ? createPortal(bar, slot) : bar;
};

export default SaveBar;
