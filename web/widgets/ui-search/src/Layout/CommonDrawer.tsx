import { Drawer } from 'antd';
import { useCallback, useEffect, useRef, type FC, type ReactNode } from 'react';

const BASE_WRAPPER_CLASS = '!overflow-hidden !rounded-12px !shadow-[0_2px_20px_rgba(0,0,0,0.1)] !dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)]';
const BASE_BODY_CLASS = '!rounded-12px';

export interface CommonDrawerProps {
  open?: boolean;
  onClose?: () => void;
  getContainer?: () => HTMLElement | null;
  placement?: 'left' | 'right';
  size?: number;
  autoFocus?: boolean;
  destroyOnHidden?: boolean;
  clickOutsideToClose?: boolean;
  classNames?: {
    wrapper?: string;
    body?: string;
  };
  children?: ReactNode;
}

const CommonDrawer: FC<CommonDrawerProps> = ({
  open,
  onClose,
  getContainer,
  placement = 'left',
  size = 260,
  autoFocus = false,
  destroyOnHidden = false,
  clickOutsideToClose = true,
  classNames: customClassNames,
  children,
}) => {
  const drawerRef = useRef<HTMLDivElement>(null);

  const handleClickOutside = useCallback(
    (e: MouseEvent | TouchEvent) => {
      if (drawerRef.current && !drawerRef.current.contains(e.target as Node)) {
        onClose?.();
      }
    },
    [onClose]
  );

  useEffect(() => {
    if (!open || !clickOutsideToClose) return;
    const container = getContainer?.() ?? document;
    container.addEventListener('mousedown', handleClickOutside as EventListener);
    container.addEventListener('touchstart', handleClickOutside as EventListener);
    return () => {
      container.removeEventListener('mousedown', handleClickOutside as EventListener);
      container.removeEventListener('touchstart', handleClickOutside as EventListener);
    };
  }, [open, clickOutsideToClose, handleClickOutside, getContainer]);

  return (
    <Drawer
      placement={placement}
      open={open}
      onClose={onClose}
      closeIcon={null}
      getContainer={getContainer as (() => HTMLElement) | undefined}
      push={false}
      mask={false}
      destroyOnHidden={destroyOnHidden}
      classNames={{
        wrapper: `${BASE_WRAPPER_CLASS} ${customClassNames?.wrapper ?? ''}`.trim(),
        body: `${BASE_BODY_CLASS} ${customClassNames?.body ?? ''}`.trim(),
      }}
      size={size}
      autoFocus={autoFocus}
    >
      <div ref={drawerRef} className="h-full flex flex-col overflow-hidden">
        {children}
      </div>
    </Drawer>
  );
};

export default CommonDrawer;
