import { useEffect, useState } from "react";
import { Logo } from "./icons/Logo";
import { isAlt, isAppleDevice, isCtrl } from "./utils";

const CTRL_KEY_DEFAULT = "Ctrl";
const CTRL_KEY_APPLE = "⌘";
const ALT_KEY_DEFAULT = "Alt";
const ALT_KEY_APPLE = "Option";

export const DocSearchButton = ({
  onClick,
  hotKeys,
  buttonText = "Search whatever you want..."
}) => {

  const [ctrlKey, setCtrlKey] = useState(null);
  const [altKey, setAltKey] = useState(null);

  useEffect(() => {
    if (typeof navigator !== "undefined") {
      if (isAppleDevice()) {
        setCtrlKey(CTRL_KEY_APPLE);
        setAltKey(ALT_KEY_APPLE);
      } else {
        setCtrlKey(CTRL_KEY_DEFAULT);
        setAltKey(ALT_KEY_DEFAULT);
      }
    }
  }, [])

  return (
    <button
      type="button"
      className="docsearch-btn"
      onClick={onClick}
      aria-label={buttonText}
    >
      <span className="docsearch-btn-icon-container">
        <Logo className="docsearch-btn-icon" />
      </span>
      <span className="docsearch-btn-placeholder"> {buttonText} </span>
      {hotKeys && hotKeys.length > 0 && (
        <span className="docsearch-btn-keys">
          {
            hotKeys[0].split("+").map((k) => (
              <kbd key={k} className="docsearch-btn-key">
                {isCtrl(k)
                  ? ctrlKey
                  : isAlt(k)
                    ? altKey
                    : k[0].toUpperCase() + k.slice(1)}
              </kbd>
            ))
          }
        </span>
      )}
    </button>
  );
};
