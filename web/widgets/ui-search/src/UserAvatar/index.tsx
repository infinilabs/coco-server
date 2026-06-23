import { CircleUser, LogOut, RefreshCw, SquareArrowOutUpRight } from "lucide-react";
import { useEffect, useRef, useState, type FC } from "react";
import { useTranslation } from 'react-i18next';
import { Button, Popover } from "antd";

interface LoginInfo {
  name?: string;
  email?: string;
}

interface UserAvatarProps {
  settings?: Record<string, any>;
  getProfile?: (callback: (data: LoginInfo) => void) => void;
  onLogout?: (refetch: () => void) => void;
  apiConfig?: { endpoint?: string; [key: string]: any };
}

const UserAvatar: FC<UserAvatarProps> = (props) => {

  const { settings, getProfile, onLogout, apiConfig } = props;

  const [loginInfo, setLoginInfo] = useState<LoginInfo | undefined>(); 
  const { t } = useTranslation();
  const hasFetched = useRef(false);

  const fetchLoginInfo = () => {
    getProfile?.((data) => {
      setLoginInfo(data);
    });
  }

  useEffect(() => {
    if (hasFetched.current) return;
    hasFetched.current = true;
    fetchLoginInfo();
  }, [])

  return (
    <div className="flex items-center relative text-sm">
      <Popover
        getPopupContainer={(trigger) => trigger.parentElement!}
        content={(
          <div className="p-3">
            <div className="flex items-center justify-between mb-2">
              <span>{t('labels.accountInfo')}</span>

              <Button 
                className="!w-24px !h-24px" 
                classNames={{
                  icon: 'flex items-center'
                }}
                color="primary" 
                variant="text" 
                icon={<RefreshCw size={12} />} 
                onClick={() => {
                  fetchLoginInfo();
                }}
              />
            </div>

            <div className="py-2">
              {loginInfo ? (
                <div className="flex justify-between items-center gap-3">
                  <div className="flex items-center gap-3">
                    <Button
                      className="rounded-full border-[#F0F0F0] dark:border-[#303030]"
                      icon={<CircleUser size={16} className="!text-[var(--ant-color-text-secondary)]" />}
                    />

                    <div className="flex flex-col">
                      <span>{loginInfo?.name}</span>
                      <span className="text-[#999] dark:text-[#666]">{loginInfo?.email}</span>
                    </div>
                  </div>

                  <Button 
                    className="!w-24px !h-24px" 
                    classNames={{
                      icon: 'flex items-center'
                    }}
                    color="primary" 
                    variant="text"
                    icon={<LogOut size={12} />} 
                    onClick={() => {
                      onLogout?.(fetchLoginInfo);
                    }}
                  />
                </div>
              ) : (
                <div className="flex flex-col items-center gap-3">
                  <span className="text-[#999] dark:text-[#666]">
                    {settings?.guest?.enabled ? t('labels.guestTip') : t('labels.pleaseLogin')}
                  </span>

                  <Button  
                    color="primary"  
                    variant="solid"
                    onClick={() => {
                      apiConfig?.endpoint && window.open(`${apiConfig.endpoint}${apiConfig.endpoint.endsWith('/') ? '#/login' : '/#/login'}`);
                    }}
                  >
                    <span>{t('labels.login')}</span>

                    <SquareArrowOutUpRight size={16} />
                  </Button>
                </div>
              )}
            </div>
          </div>
        )}
        trigger="click"
        arrow={false}
        classNames={{
          container: '!p-0',
          content: '!p-0',
        }}
      >
        <Button
          className="rounded-12px border-[#F0F0F0] dark:border-[#303030]"
          icon={<CircleUser size={16} className="!text-[var(--ant-color-text-secondary)]" />}
        />
      </Popover>
    </div>
  );
};

export default UserAvatar;
