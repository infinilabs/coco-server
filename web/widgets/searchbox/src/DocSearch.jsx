import { useEffect, useMemo, useRef, useState } from 'react';

import { DocSearchButton } from './DocSearchButton';
import { DocSearchFloatButton } from './DocSearchFloatButton';
import { DocSearchModal } from './DocSearchModal';
import { isAlt, isAppleDevice, isCtrl, isMeta } from './utils';

const DEFAULT_HOTKEYS = ['ctrl+/'];

export const DocSearch = props => {
  const { hotKeys = DEFAULT_HOTKEYS, id, server, token } = props;

  const [isOpen, setIsOpen] = useState(false);
  const [initialQuery, setInitialQuery] = useState();
  const [settings, setSettings] = useState();

  const [triggerBtnType, setTriggerBtnType] = useState('embedded');
  const onOpen = () => setIsOpen(true);
  const onClose = () => {
    setIsOpen(false);
    setTriggerBtnType();
  };
  const onInput = query => setInitialQuery(query);
  const onClick = type => {
    const selectedText = window.getSelection();
    if (selectedText) setInitialQuery(selectedText.toString());
    setTriggerBtnType(type);
    setIsOpen(true);
  };

  function isEditingContent(event) {
    const element = event.target;
    const tagName = element.tagName;

    return element.isContentEditable || tagName === 'INPUT' || tagName === 'SELECT' || tagName === 'TEXTAREA';
  }

  const currentHotkeys = useMemo(() => {
    let formatHotKey = settings?.hotkey;
    if (!isAppleDevice() && formatHotKey?.includes('meta')) {
      formatHotKey = formatHotKey.replace('meta', 'ctrl');
    }
    return formatHotKey ? [formatHotKey] : hotKeys;
  }, [hotKeys, settings?.hotkey]);

  function isHotKey(event) {
    const modsAndkeys = currentHotkeys && currentHotkeys.map(k => k.toLowerCase().split('+'));

    if (modsAndkeys) {
      return modsAndkeys.some(modsAndkeys => {
        // if hotkey is a single character, we only react if modal is not open
        if (
          modsAndkeys.length === 1 &&
          event.key.toLowerCase() === modsAndkeys[0] &&
          !event.ctrlKey &&
          !event.altKey &&
          !event.shiftKey &&
          !isEditingContent(event) &&
          !isOpen
        ) {
          return true;
        }

        // modifiers and key
        if (modsAndkeys.length > 1) {
          const key = modsAndkeys[modsAndkeys.length - 1];

          if (event.key.toLowerCase() !== key) return false;

          const ctrl = event.ctrlKey == modsAndkeys.some(isCtrl);
          const shift = event.shiftKey == modsAndkeys.includes('shift');
          const alt = event.altKey == modsAndkeys.some(isAlt);
          const meta = !isAppleDevice() || event.metaKey == modsAndkeys.some(isMeta);

          return ctrl && shift && alt && meta;
        }

        return false;
      });
    }

    return false;
  }

  function onKeyDown(e) {
    if ((e.key === 'Escape' && isOpen) || isHotKey(e)) {
      e.preventDefault();
      if (isOpen) {
        onClose();
      } else if (!document.body.classList.contains('infini__searchbox--active')) {
        // We check that no other DocSearch modal is showing before opening
        // another one.
        const selectedText = window.getSelection();
        if (selectedText) onInput(selectedText.toString());
        onOpen();
      }
    }
  }

  async function fetchSettings(server, id, token) {
    if (!server || !id || !token) return;
    fetch(`${server}/integration/${id}`, {
      headers: {
        'APP-INTEGRATION-ID': id,
        'Content-Type': 'application/json',
        'X-API-TOKEN': token
      },
      method: 'GET'
    })
      .then(response => response.json())
      .then(result => {
        if (result?._source) {
          setSettings(result?._source);
        }
      })
      .catch(error => console.log('error', error));
  }

  function renderButton(settings) {
    const { options, type } = settings || {};
    const searchButton = (
      <DocSearchButton
        buttonText={options?.placeholder}
        hotKeys={currentHotkeys}
        onClick={() => onClick('embedded')}
      />
    );
    const floatButton = <DocSearchFloatButton onClick={() => onClick('floating')} />;
    if (type === 'floating') {
      return floatButton;
    }
    if (type === 'all') {
      return (
        <>
          {searchButton}
          {floatButton}
        </>
      );
    }
    if (type === 'embedded') {
      return searchButton;
    }
    return null;
  }

  useEffect(() => {
    window.removeEventListener('keydown', onKeyDown);
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [isOpen, currentHotkeys]);

  useEffect(() => {
    fetchSettings(server, id, token);
  }, [server, id, token]);

  useEffect(() => {
    const body = document.body;
    body.style.overflow = isOpen ? 'hidden' : 'auto';
  }, [isOpen]);

  return (
    <div
      data-theme={settings?.appearance?.theme}
      id="infini__searchbox"
    >
      {renderButton(settings)}
      {isOpen && (
        <DocSearchModal
          initialQuery={initialQuery}
          server={server}
          settings={settings}
          triggerBtnType={triggerBtnType}
          onClose={onClose}
        />
      )}
    </div>
  );
};
