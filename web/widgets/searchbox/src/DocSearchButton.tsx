import { useEffect, useRef, useState } from 'react';

import SearchLogo from './icons/search-logo.svg?react';
import { isAlt, isAppleDevice, isCtrl, isMeta } from './utils';

const CTRL_KEY_DEFAULT = 'Ctrl';
const CTRL_KEY_APPLE = '⌃';
const ALT_KEY_DEFAULT = 'Alt';
const ALT_KEY_APPLE = '⌥';
const META_KEY_APPLE = '⌘';

interface DocSearchButtonProps {
  settings?: any;
  hotKeys?: string[];
  onClick?: () => void;
}

export const DocSearchButton: React.FC<DocSearchButtonProps> = ({ settings, hotKeys, onClick }) => {
  const [ctrlKey, setCtrlKey] = useState<string | null>(null);
  const [altKey, setAltKey] = useState<string | null>(null);
  const [metaKey, setMetaKey] = useState<string | null>(null);
  const resizeObserverRef = useRef<ResizeObserver | null>(null)
  const btnRef = useRef<HTMLButtonElement>(null)
  const [displayState, setDisplayState] = useState({
    key: true,
    placeholder: true,
  });

  const { options } = settings || {};

  useEffect(() => {
    if (typeof navigator !== 'undefined') {
      if (isAppleDevice()) {
        setCtrlKey(CTRL_KEY_APPLE);
        setAltKey(ALT_KEY_APPLE);
        setMetaKey(META_KEY_APPLE);
      } else {
        setCtrlKey(CTRL_KEY_DEFAULT);
        setAltKey(ALT_KEY_DEFAULT);
      }
    }
  }, []);

  useEffect(() => {
    resizeObserverRef.current?.disconnect();
    const element = btnRef.current;
    if (element) {
      resizeObserverRef.current = new ResizeObserver((entries) => {
        for (const entry of entries) {
          const target = entry.target as HTMLElement;
          if (target.offsetWidth < 48) {
            setDisplayState({
              key: false,
              placeholder: false,
            })
          } else if (target.offsetWidth < 120) {
            setDisplayState({
              key: false,
              placeholder: true,
            })
          } else {
            setDisplayState({
              key: true,
              placeholder: true,
            })
          }
        }
      });

      resizeObserverRef.current.observe(element);

      return () => {
        resizeObserverRef.current?.disconnect();
      };
    }
  }, [options?.embedded_placeholder, hotKeys]);

  return (
    <button
      className="infini__searchbox-btn"
      type="button"
      onClick={onClick}
      ref={btnRef}
    >
      <span className="infini__searchbox-btn-icon-container">
        { options?.embedded_icon ? (
          <img src={options?.embedded_icon}/>
        ) : <SearchLogo className="infini__searchbox-btn-icon" /> }
      </span>
      {
        displayState.placeholder && <span className="infini__searchbox-btn-placeholder"> {options?.embedded_placeholder || 'Search...'} </span>
      }
      {
        displayState.key && (
          hotKeys && hotKeys.length > 0 && (
            <span className="infini__searchbox-btn-keys">
              {hotKeys[0].split('+').map(k => (
                <kbd
                  className="infini__searchbox-btn-key"
                  key={k}
                >
                  {isMeta(k) ? metaKey : isCtrl(k) ? ctrlKey : isAlt(k) ? altKey : k[0].toUpperCase() + k.slice(1)}
                </kbd>
              ))}
            </span>
          )
        )
      }
    </button>
  );
};
