import { DocSearchButton } from "./DocSearchButton";
import { DocSearchModal } from "./DocSearchModal";
import { useEffect, useState } from "react";
import { isAlt, isAppleDevice, isCtrl, isMeta } from "./utils";
import { DocSearchFloatButton } from "./DocSearchFloatButton";

const DEFAULT_HOTKEYS = ["ctrl+/"];

export const DocSearch = (props) => {
  const { hotKeys = DEFAULT_HOTKEYS, server, id, token } = props;

  const [isOpen, setIsOpen] = useState(false);
  const [initialQuery, setInitialQuery] = useState();
  const [settings, setSettings] = useState()

  const onOpen = () => setIsOpen(true);
  const onClose = () => setIsOpen(false);
  const onInput = (query) => setInitialQuery(query);
  const onClick = () => {
    const selectedText = window.getSelection();
    if (selectedText) setInitialQuery(selectedText.toString());
    setIsOpen(true);
  };

  function isEditingContent(event) {
    const element = event.target;
    const tagName = element.tagName;

    return (
      element.isContentEditable ||
      tagName === "INPUT" ||
      tagName === "SELECT" ||
      tagName === "TEXTAREA"
    );
  }

  function isHotKey(event) {
    const modsAndkeys =
      hotKeys && hotKeys.map((k) => k.toLowerCase().split("+"));

    if (modsAndkeys) {
      return modsAndkeys.some((modsAndkeys) => {
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

          const ctrl =
            (isAppleDevice() ? event.metaKey : event.ctrlKey) ==
            modsAndkeys.some(isCtrl);
          const shift = event.shiftKey == modsAndkeys.includes("shift");
          const alt = event.altKey == modsAndkeys.some(isAlt);
          const meta =
            !isAppleDevice() && event.metaKey == modsAndkeys.some(isMeta);

          return ctrl && shift && alt && meta;
        }

        return false;
      });
    }

    return false;
  }

  function onKeyDown(e) {
    if ((e.key === "Escape" && isOpen) || isHotKey(e)) {
      e.preventDefault();
      if (isOpen) {
        onClose();
      } else if (!document.body.classList.contains("infini__searchbox--active")) {
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
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          "X-API-TOKEN": token, 
          "APP-INTEGRATION-ID": id
        },
      }).then(response => response.json())
        .then(result => {
          if (result?._source) {
            setSettings(result?._source)
          }
        })
        .catch(error => console.log('error', error));
  }

  function renderButton(settings) {
    const { type, enabled_module } = settings || {};
    const searchButton = (
      <DocSearchButton
        buttonText={enabled_module?.search?.placeholder}
        hotKeys={hotKeys}
        onClick={onClick}
      />
    )
    const floatButton = (
      <DocSearchFloatButton onClick={onClick}/>
    )
    if (type === 'floating') {
      return floatButton
    }
    if (type === 'all') {
      return (
        <>
          {searchButton}
          {floatButton}
        </>
      )
    }
    if (type === 'embedded') {
      return searchButton
    }
    return null
  }

  useEffect(() => {
    window.removeEventListener("keydown", onKeyDown)
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [isOpen])

  useEffect(() => {
    fetchSettings(server, id, token)
  }, [server, id, token])

  return (
    <div id="infini__searchbox" data-theme={settings?.appearance?.theme}>
      {renderButton(settings)}
      {isOpen && (
          <DocSearchModal
            server={server}
            settings={settings}
            initialQuery={initialQuery}
            onClose={onClose}
          />
      )}
    </div>
  );
};
