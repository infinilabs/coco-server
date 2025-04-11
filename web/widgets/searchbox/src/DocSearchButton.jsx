import { useEffect, useState } from 'react';

import SearchLogo from './icons/search-logo.svg';
import { isAlt, isAppleDevice, isCtrl, isMeta } from './utils';

const CTRL_KEY_DEFAULT = 'Ctrl';
const CTRL_KEY_APPLE = '⌃';
const ALT_KEY_DEFAULT = 'Alt';
const ALT_KEY_APPLE = '⌥';
const META_KEY_APPLE = '⌘';

export const DocSearchButton = ({ settings, hotKeys, onClick }) => {
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

  const { options } = settings || {};

  return (
    <button
      className="infini__searchbox-btn"
      type="button"
      onClick={onClick}
    >
      <span className="infini__searchbox-btn-icon-container">
        { options?.embedded_icon ? (
          <img src={options?.embedded_icon}/>
        ) : <SearchLogo className="infini__searchbox-btn-icon" /> }
      </span>
      <span className="infini__searchbox-btn-placeholder"> {options?.embedded_placeholder || 'Search...'} </span>
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
