import { DocSearchButton } from "./DocSearchButton";
import { DocSearchModal } from "./DocSearchModal";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { isAlt, isAppleDevice, isCtrl, isMeta } from "./utils";
import { DocSearchFloatButton } from "./DocSearchFloatButton";
import { createRoot } from 'react-dom/client';

const DEFAULT_HOTKEYS = ["ctrl+/"];
const DARK_MODE_MEDIA_QUERY = '(prefers-color-scheme: dark)'

export const DocSearch = (props) => {
  const { hotKeys = DEFAULT_HOTKEYS, server, id, token, linkHref } = props;

  const [isOpen, setIsOpen] = useState(false);
  const [initialQuery, setInitialQuery] = useState();
  const [settings, setSettings] = useState()
  const [theme, setTheme] = useState(window.matchMedia && window.matchMedia(DARK_MODE_MEDIA_QUERY).matches ? 'dark' : 'light')

  const [shadowLoading, setShadowLoading] = useState(true)
  const shadowRef = useRef(null)
  const modelRef = useRef(null)
  const floatBtnRef = useRef(null)

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

  const onKeyDown = useCallback((e) => {
    if ((e.key === "Escape" && isOpen) || isHotKey(e)) {
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
  }, [isOpen, currentHotkeys])

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

  function renderShadow(linkHref) {
    const container = document.createElement("div");
    document.body.appendChild(container)
    const shadow = container.attachShadow({ mode: "open" });
    shadowRef.current = shadow;
    if (linkHref) {
      const linkElement = document.createElement("link");
      linkElement.rel = "stylesheet";
      linkElement.href = linkHref;
      shadow.appendChild(linkElement);
      linkElement.onload = async () => {
        setShadowLoading(false)
      }
    } else {
      setShadowLoading(false)
    }
  }

  function renderModal(server, settings, triggerBtnType, theme, isOpen) {
    if (!shadowRef.current) return;

    if (modelRef.current) modelRef.current.remove()

    const props = {
      server,
      settings,
      onClose,
      triggerBtnType,
      theme,
      isOpen
    }
    const wrapper = document.createElement("div");
    shadowRef.current.appendChild(wrapper);
    modelRef.current = wrapper
    const root = createRoot(wrapper);
    root.render(<DocSearchModal {...props} />);
  }

  function renderFloatButton(theme, settings) {
    if (!shadowRef.current || !['floating', 'all'].includes(settings?.type)) return;

    if (floatBtnRef.current) floatBtnRef.current.remove()

    const props = {
      theme,
      settings,
      onClick: () => onClick('floating')
    }
    const wrapper = document.createElement("div");
    shadowRef.current.appendChild(wrapper);
    floatBtnRef.current = wrapper
    const root = createRoot(wrapper);
    root.render(<DocSearchFloatButton {...props}/>);
  }

  function renderButton(settings) {
    const { type } = settings || {};
    if (['embedded', 'all'].includes(type)) {
      return (
        <DocSearchButton
          settings={settings}
          hotKeys={currentHotkeys}
          onClick={() => onClick('embedded')}
        />
      )
    }
    return null
  }

  function onSystemThemeChange(e) {
    setTheme(e.matches ? 'dark' : 'light')
  }

  useEffect(() => {
    window.removeEventListener("keydown", onKeyDown)
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [onKeyDown])

  useEffect(() => {
    fetchSettings(server, id, token);
  }, [server, id, token]);

  useEffect(() => {
    const body = document.body;
    body.style.overflow = isOpen ? 'hidden' : 'auto';
    if (isOpen) {
      document.body.classList.add("infini__searchbox--active")
    } else {
      document.body.classList.remove("infini__searchbox--active")
    }
  }, [isOpen])

  useEffect(() => {
    if (settings?.appearance?.theme === 'auto') {
      setTheme(window.matchMedia && window.matchMedia(DARK_MODE_MEDIA_QUERY).matches ? 'dark' : 'light')
      window.matchMedia(DARK_MODE_MEDIA_QUERY).addEventListener('change', onSystemThemeChange);
    } else {
      setTheme(settings?.appearance?.theme)
    }
    return () => {
      if (settings?.appearance?.theme === 'auto') {
        window.matchMedia(DARK_MODE_MEDIA_QUERY).removeEventListener('change', onSystemThemeChange)
      }
    }
  }, [settings?.appearance?.theme])

  useEffect(() => {
    renderShadow(linkHref)
  }, [linkHref])

  useEffect(() => {
    if (!shadowLoading) {
      renderModal(server, settings, triggerBtnType, theme, isOpen)
    }
  }, [shadowLoading, server, settings, triggerBtnType, theme, isOpen])

  useEffect(() => {
    if (!shadowLoading) {
      renderFloatButton(theme, settings)
    }
  }, [shadowLoading, theme, theme])

  return (
    <div
      data-theme={theme}
      id="infini__searchbox"
    >
      {renderButton(settings)}
    </div>
  );
};
