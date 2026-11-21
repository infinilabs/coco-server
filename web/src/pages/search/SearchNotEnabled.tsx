import { Button, Result, Space, Typography } from 'antd';
import { SearchOutlined, SettingOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';

import { useAppSelector } from '@/hooks/business/useStore';
import { selectUserInfo } from '@/store/slice/auth';

interface Props {
  /** hide the "go to settings" button when the reader cannot reach it */
  canManage?: boolean;
}

/**
 * Shown on /search when the operator has not enabled the AI search component
 * (搜索设置 · 基础与组件: 启用状态 / 组件). Replaces the blank 404 the router
 * used to produce for this case with an actionable pointer to the setting.
 */
const SearchNotEnabled: FC<Props> = memo(({ canManage }) => {
  const { t } = useTranslation();
  const nav = useNavigate();
  const userInfo = useAppSelector(selectUserInfo);

  const isAdmin = canManage ?? Boolean(
    userInfo?.roles?.includes('admin') || userInfo?.permissions?.includes('coco#system/read')
  );

  return (
    <div className='relative size-full min-h-520px flex-col-center overflow-hidden'>
      <div className='pointer-events-none absolute -top-100px -left-80px size-400px rounded-full bg-primary/10 blur-3xl dark:bg-primary/15' />
      <div className='pointer-events-none absolute -right-80px -bottom-120px size-420px rounded-full bg-info/10 blur-3xl dark:bg-info/15' />
      <Result
        className='z-1'
        icon={<SearchOutlined className='text-primary text-72px' />}
        title={<Typography.Title level={3} className='!mb-0'>{t('page.search_settings.not_enabled.title')}</Typography.Title>}
        subTitle={
          <Typography.Text type='secondary' className='max-w-480px'>
            {isAdmin
              ? t('page.search_settings.not_enabled.desc_admin')
              : t('page.search_settings.not_enabled.desc_user')}
          </Typography.Text>
        }
        extra={
          <Space>
            <Button onClick={() => nav('/')}>{t('common.backToHome')}</Button>
            {isAdmin && (
              <Button
                type='primary'
                icon={<SettingOutlined />}
                onClick={() => nav('/search-settings')}
              >
                {t('page.search_settings.not_enabled.go_settings')}
              </Button>
            )}
          </Space>
        }
      />
    </div>
  );
});

export default SearchNotEnabled;
