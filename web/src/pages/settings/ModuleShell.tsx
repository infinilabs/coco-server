import { Card } from 'antd';
import { type ReactNode, useState } from 'react';

import './index.scss';
import PageHeaderTitle from '@/components/stateless/common/PageHeaderTitle';
import { SaveBarSlotContext } from './modules/SaveBar';

interface Props {
  children: ReactNode;
  /** page description shown directly under the title (one header unit) */
  description?: string;
  icon?: ReactNode;
  title?: string;
}

/**
 * Page shell for settings modules promoted to first-class routes (search
 * settings / wiki settings / engine AI). These modules were built to render
 * inside the settings page's card, which provided the white background —
 * without this shell they float on the bare layout canvas.
 *
 * The optional title/description render as a stacked header block in the Card
 * head (same unit as every other first-level page). The slot div after the
 * Card is the SaveBar portal target: modules' save bars render OUTSIDE the
 * white card on the page canvas, matching the ontology page's save bar.
 */
export default function ModuleShell({ children, description, icon, title }: Props) {
  const [slot, setSlot] = useState<HTMLElement | null>(null);

  return (
    <SaveBarSlotContext.Provider value={slot}>
      <Card
        styles={{ body: { padding: 0 } }}
        title={title ? <PageHeaderTitle description={description} icon={icon} title={title} /> : undefined}
      >
        <div className="settings-tabs-content">{children}</div>
      </Card>
      <div ref={setSlot} style={{ display: 'contents' }} />
    </SaveBarSlotContext.Provider>
  );
}
