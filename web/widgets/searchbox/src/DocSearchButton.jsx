import { useEffect, useState } from 'react';

import { Logo } from './icons/Logo';
import { isAlt, isAppleDevice, isCtrl, isMeta } from './utils';

const CTRL_KEY_DEFAULT = 'Ctrl';
const CTRL_KEY_APPLE = '⌃';
const ALT_KEY_DEFAULT = 'Alt';
const ALT_KEY_APPLE = '⌥';
const META_KEY_APPLE = '⌘';

export const DocSearchButton = ({ buttonText = 'Search whatever you want...', hotKeys, onClick }) => {
  const [ctrlKey, setCtrlKey] = useState(null);
  const [altKey, setAltKey] = useState(null);
  const [metaKey, setMetaKey] = useState(null);

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

  return (
    <button
      aria-label={buttonText}
      className="infini__searchbox-btn"
      type="button"
      onClick={onClick}
    >
      <span className="infini__searchbox-btn-icon-container">
        <Logo className="infini__searchbox-btn-icon" />
      </span>
      <span className="infini__searchbox-btn-placeholder"> {buttonText} </span>
      {hotKeys && hotKeys.length > 0 && (
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
      )}
    </button>
  );
};
