import { useState } from "react";
import { useEffect } from "react";
import SearchChat  from '@infinilabs/search-chat';

export const DocSearchModal = ({
  server,
  settings,
  onClose,
  triggerBtnType
}) => {
  // We rely on a CSS property to set the modal height to the full viewport height
  // because all mobile browsers don't compute their height the same way.
  // See https://css-tricks.com/the-trick-to-viewport-units-on-mobile/

  const [isPinned, setIsPinned] = useState(false)

  let modalRef;
  function setFullViewportHeight() {
    if (modalRef) {
      const vh = window.innerHeight * 0.01;
      modalRef.style.setProperty("--infini__searchbox-vh", `${vh}px`);
    }
  }
  useEffect(() => {
    document.body.classList.add("infini__searchbox--active")
    setFullViewportHeight();
    window.addEventListener("resize", setFullViewportHeight);
    return () => {
      document.body.classList.remove("infini__searchbox--active")
      window.removeEventListener("resize", setFullViewportHeight)
    }
  }, [])

  const { id, type, token, enabled_module = {}, appearance = {} } = settings;
  const { search, ai_chat, features } = enabled_module

  const hasModules = []
  if (search?.enabled) {
    hasModules.push('search')
  }
  if (ai_chat?.enabled) {
    hasModules.push('chat')
  }
  
  let defaultModule = 'search'
  if (type === 'embedded') {
    defaultModule = 'search'
  } else if (type === 'floating') {
    defaultModule = 'chat'
  } else if (type === 'all') {
    if (triggerBtnType === 'embedded') {
      defaultModule = 'search'
    } else if (triggerBtnType === 'floating') {
      defaultModule = 'chat'
    }
  }

  return (
    <div
      className="infini__searchbox-modal-container"
      role="button"
      tabIndex={0}
      ref={modalRef}
      onMouseDown={(e) => e.target === e.currentTarget && onClose && !isPinned && onClose()}
    >
      <div className="infini__searchbox-modal">
        <SearchChat
          serverUrl={server}
          headers={{
            "X-API-TOKEN": token, 
            "APP-INTEGRATION-ID": id
          }}
          width={680}
          height={590}
          hasModules={hasModules}
          searchPlaceholder={search?.placeholder || 'Search whatever you want...'}
          chatPlaceholder={ai_chat?.placeholder || 'Ask whatever you want...'}
          hasFeature={features || []}
          theme={appearance?.theme}
          showChatHistory={features?.includes('chat_history')}
          setIsPinned={setIsPinned}
          defaultModule={defaultModule}
        />
      </div>
    </div>
  );
};