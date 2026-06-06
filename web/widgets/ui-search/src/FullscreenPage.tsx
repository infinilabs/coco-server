import { useEffect, useRef, useState } from "react";

import Fullscreen from "./Fullscreen";
import { isEmpty } from "lodash";
import UserAvatar from "./UserAvatar";
import { I18nextProvider } from 'react-i18next';
import i18nInstance from "./i18n";

interface FullscreenPageProps {
  language?: string;
  settings?: Record<string, any>;
  onSearch?: (...args: any[]) => void;
  queryParams?: Record<string, any>;
  setQueryParams?: (params: any) => void;
  onLogoClick?: () => void;
  apiConfig?: Record<string, any>;
  getProfile?: () => void;
  onLogout?: () => void;
  showTopAction?: boolean;
  rightMenuWidth?: number;
  [key: string]: any;
}

const FullscreenPage = (props: FullscreenPageProps) => {
  const { language, settings, onSearch, queryParams, setQueryParams, onLogoClick, apiConfig, getProfile, onLogout, showTopAction } = props;

  const [isHome, setIsHome] = useState(true);
  const isHomeRef = useRef(true);
  const topActionsRef = useRef<HTMLDivElement>(null)
  const [rightMenuWidth, setRightMenuWidth] = useState(0);

  useEffect(() => {
    i18nInstance.changeLanguage(language);
  }, [language]);

  useEffect(() => {
    const element = topActionsRef.current;

    if (!element || !showTopAction) {
      return;
    }

    const updateRightMenuWidth = () => {
      const width = Math.ceil(element.getBoundingClientRect().width);
      setRightMenuWidth(width > 0 ? width : 0);
    };

    updateRightMenuWidth();

    const observer = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(updateRightMenuWidth) : null;

    observer?.observe(element);
    window.addEventListener('resize', updateRightMenuWidth);

    return () => {
      observer?.disconnect();
      window.removeEventListener('resize', updateRightMenuWidth);
    };
  }, [showTopAction]);

  return (
    <I18nextProvider i18n={i18nInstance}>
      <Fullscreen
        {...props}
        isHome={queryParams?.query || !isEmpty(queryParams?.filter) || !isEmpty(queryParams?.aggfilter) ? false : isHome}
        onSearch={(...args: any[]) => {
          if (isHomeRef.current) {
            setIsHome(false);
            isHomeRef.current = false;
          }
          onSearch?.(...args);
        }}
        onLogoClick={() => {
          setIsHome(true)
          onLogoClick && onLogoClick()
        }}
        rightMenuWidth={props.rightMenuWidth || rightMenuWidth}
      />
      {
        showTopAction && (
          <div ref={topActionsRef} style={{ top: queryParams?.mode === 'chat' ? 8 : 16 }} className="pl-16px absolute right-16px h-48px z-1002 flex items-center">
            <UserAvatar 
              settings={settings} 
              apiConfig={apiConfig} 
              getProfile={getProfile}
              onLogout={onLogout}
            />
          </div>
        )
      }
    </I18nextProvider>
  );
};

export default FullscreenPage;
