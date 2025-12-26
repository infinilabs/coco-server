import { useLoading } from '@sa/hooks';
import { Spin } from 'antd';

import { generateRandomString } from '@/utils/common';
import UserAvatar from '@/layouts/modules/global-header/components/UserAvatar';
import { localStg } from '@/utils/storage';
import { getApiBaseUrl } from '@/service/request';
import { getThemeSettings } from '@/store/slice/theme';
import { configResponsive } from 'ahooks';
import { selectUserInfo } from '@/store/slice/auth';

const uuid = `integration-${generateRandomString(8)}`

configResponsive({ sm: 640 });

export function Component() {
  const { endLoading, loading, startLoading } = useLoading();
  const containerRef = useRef(null)
  const isFirstRef = useRef(true)
  const themeRef = useRef('')
  
  const responsive = useResponsive();

  const userInfo = useAppSelector(selectUserInfo);
  const { themeScheme } = useAppSelector(getThemeSettings);

  const providerInfo = localStg.get('providerInfo') || {}

  const { search_settings } = providerInfo;

  const isMobile = !responsive.sm;

  const clearAll = () => {
    const originalDiv = containerRef.current;
    if (!originalDiv) return; 

    const newDiv = document.createElement('div');

    newDiv.id = originalDiv.id; 
    newDiv.className = originalDiv.className; 
    newDiv.ref = originalDiv.ref;

    Array.from(originalDiv.attributes).forEach(attr => {
      if (!['id', 'class', 'ref'].includes(attr.name)) { 
        newDiv.setAttribute(attr.name, attr.value);
      }
    });

    originalDiv.parentNode?.replaceChild(newDiv, originalDiv);

    containerRef.current = newDiv;
  };

  useEffect(() => {
    if (search_settings?.integration && search_settings.enabled && containerRef.current && (isFirstRef.current || themeRef.current !== themeScheme)) {
      isFirstRef.current = false
      themeRef.current = themeScheme
      startLoading();
      import(/* @vite-ignore */ `${getApiBaseUrl()}integration/${search_settings?.integration}/widget`)
        .then(module => {
          clearAll()
          module?.fullscreen && module.fullscreen({ 
            container: `#${uuid}`, 
            rightMenuWidth: userInfo ? 90 : 136, 
            parentTheme: themeScheme,
          });
          endLoading();
        })
        .catch(err => {
          console.log(err)
          endLoading();
        });
    }
  }, [search_settings?.integration, search_settings.enabled, uuid, containerRef.current, themeScheme, userInfo])

  return (
    <Spin spinning={loading}>
      <div ref={containerRef} id={uuid}></div>
      <div className="absolute right-12px top-16px z-1 flex-y-center justify-end">
        {
          isMobile ? (
            <>
                <ThemeSchemaSwitch className="px-12px" />
                <UserAvatar className="px-8px" showHome showName={!isMobile}/>
            </>
          ) : (
            <>
                <ThemeSchemaSwitch className="px-12px" />
                <UserAvatar className="px-8px" showHome showName={!isMobile}/>
            </>
          )
        }
      </div>
    </Spin>
  );
}
