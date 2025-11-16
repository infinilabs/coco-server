import { useLoading } from '@sa/hooks';
import { Spin } from 'antd';

import { generateRandomString } from '@/utils/common';
import UserAvatar from '@/layouts/modules/global-header/components/UserAvatar';
import { localStg } from '@/utils/storage';
import { getApiBaseUrl } from '@/service/request';

const uuid = `integration-${generateRandomString(8)}`

export function Component() {
  const { endLoading, loading, startLoading } = useLoading();
  const containerRef = useRef(null)
  const isFirstRef = useRef(true)

  const providerInfo = localStg.get('providerInfo') || {}

  const { search_settings } = providerInfo;

  useEffect(() => {
    if (search_settings?.integration && search_settings.enabled && containerRef.current && isFirstRef.current) {
      isFirstRef.current = false
      startLoading();
      import(/* @vite-ignore */ `${getApiBaseUrl()}integration/${search_settings?.integration}/widget`)
        .then(module => {
          module?.fullscreen && module.fullscreen({ container: `#${uuid}`, rightMenuWidth: 72 });
          endLoading();
        })
        .catch(err => {
          console.log(err)
          endLoading();
        });
    }
  }, [search_settings?.integration, search_settings.enabled, uuid, containerRef.current])

  return (
    <Spin spinning={loading}>
      <div ref={containerRef} id={uuid}></div>
      <div className="absolute right-12px top-16px z-1 flex-y-center justify-end">
        <UserAvatar className="px-8px" showHome/>
      </div>
    </Spin>
  );
}
