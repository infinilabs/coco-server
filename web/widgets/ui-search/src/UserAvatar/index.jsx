import { CircleUser, LogOut, RefreshCw, SquareArrowOutUpRight } from "lucide-react";
import clsx from "clsx";
import { useEffect, useState } from "react";
import { Button, Popover } from "antd";

const UserAvatar = (props) => {

  const { settings, getProfile, onLogout, apiConfig } = props;

  const [loginInfo, setLoginInfo] = useState(); 

  const fetchLoginInfo = () => {
    getProfile?.((data) => {
      setLoginInfo(data);
    });
  }

  useEffect(() => {
    fetchLoginInfo();
  }, [])

  return (
    <div className="flex items-center relative text-sm">
      <Popover
        content={(
          <div className="p-3">
            <div className="flex items-center justify-between mb-2">
              <span>账户信息</span>

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
                      className="rounded-full"
                      icon={<CircleUser size={16} className="!text-[var(--ant-color-text-secondary)]" />}
                    />

                    <div className="flex flex-col">
                      <span>{loginInfo?.name}</span>
                      <span className="text-[#999]">{loginInfo?.email}</span>
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
                  <span className="text-[#999]">
                    {settings?.guest?.enabled
                      ? '游客模式，登录解锁完整体验'
                      : '请登录您的账户以开始。'
                    }
                  </span>

                  <Button  
                    color="primary"  
                    variant="solid"
                    onClick={() => {
                      apiConfig?.endpoint && window.open(`${apiConfig.endpoint}${apiConfig.endpoint.endsWith('/') ? '#/login' : '/#/login'}`);
                    }}
                  >
                    <span>登录</span>

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
          className="rounded-12px"
          icon={<CircleUser size={16} className="!text-[var(--ant-color-text-secondary)]" />}
        />
      </Popover>
    </div>
  );
};

export default UserAvatar;
