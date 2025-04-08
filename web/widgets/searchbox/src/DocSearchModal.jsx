import SearchChat from '@infinilabs/search-chat';
import { useEffect, useState } from 'react';

export const DocSearchModal = ({ onClose, server, settings, triggerBtnType }) => {
  // We rely on a CSS property to set the modal height to the full viewport height
  // because all mobile browsers don't compute their height the same way.
  // See https://css-tricks.com/the-trick-to-viewport-units-on-mobile/

  const [isPinned, setIsPinned] = useState(false);

  let modalRef;
  function setFullViewportHeight() {
    if (modalRef) {
      const vh = window.innerHeight * 0.01;
      modalRef.style.setProperty('--infini__searchbox-vh', `${vh}px`);
    }
  }
  useEffect(() => {
    document.body.classList.add('infini__searchbox--active');
    setFullViewportHeight();
    window.addEventListener('resize', setFullViewportHeight);
    return () => {
      document.body.classList.remove('infini__searchbox--active');
      window.removeEventListener('resize', setFullViewportHeight);
    };
  }, []);

  const { appearance = {}, enabled_module = {}, id, token, type } = settings;
  const { ai_chat, features, search } = enabled_module;

  const hasModules = [];
  if (search?.enabled) {
    hasModules.push('search');
  }
  if (ai_chat?.enabled) {
    hasModules.push('chat');
  }

  let defaultModule = 'search';
  if (type === 'embedded') {
    defaultModule = 'search';
  } else if (type === 'floating') {
    defaultModule = 'chat';
  } else if (type === 'all') {
    if (triggerBtnType === 'embedded') {
      defaultModule = 'search';
    } else if (triggerBtnType === 'floating') {
      defaultModule = 'chat';
    }
  }

  return (
    <div
      className="infini__searchbox-modal-container"
      ref={modalRef}
      role="button"
      tabIndex={0}
      onMouseDown={e => e.target === e.currentTarget && onClose && !isPinned && onClose()}
    >
      <div className="infini__searchbox-modal">
        <SearchChat
          chatPlaceholder={ai_chat?.placeholder || 'Ask whatever you want...'}
          defaultModule={defaultModule}
          hasFeature={features || []}
          hasModules={hasModules}
          height={590}
          searchPlaceholder={search?.placeholder || 'Search whatever you want...'}
          serverUrl={server}
          setIsPinned={setIsPinned}
          showChatHistory={features?.includes('chat_history')}
          theme={appearance?.theme}
          width={680}
          headers={{
            'APP-INTEGRATION-ID': id,
            'X-API-TOKEN': token
          }}
        />
      </div>
    </div>
  );
};
