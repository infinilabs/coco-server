import { SafetyCertificateOutlined } from '@ant-design/icons';
import PageHeaderTitle from '@/components/stateless/common/PageHeaderTitle';
import { Tabs } from "antd";

import Auth from "./modules/Auth";
import Role from "./modules/Role";

import "./index.scss";
import User from "./modules/User";
import { getApplicationSetting } from "@/store/slice/server";

export function Component() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth();

  const permissions = {
    viewAuth: hasAuth("generic#security:authorization/search"),
    viewUser: hasAuth("generic#security:user/search"),
    viewRole: hasAuth("generic#security:role/search"),
  };

  const applicationSetting = useAppSelector(getApplicationSetting);

  const onChange = (key: string) => {
    setSearchParams({ tab: key });
  };

  const items: any[] = [];

  if (permissions.viewAuth && applicationSetting?.security?.managed) {
    items.push({
      component: Auth,
      key: "auth",
      label: t(`page.auth.title`),
    });
  }

  if (permissions.viewUser && !applicationSetting?.security?.managed) {
    items.push({
      component: User,
      key: "user",
      label: t(`page.user.title`),
    });
  }

  if (permissions.viewRole) {
    items.push({
      component: Role,
      key: "role",
      label: t(`page.role.title`),
    });
  }

  // no useMemo: the component is kept alive across navigations, so the
  // tab must re-derive from searchParams on every render or a later
  // #/security?tab=role deep link stays on the first-render tab
  const activeKey = searchParams.get("tab") || items?.[0]?.key;
  const activeItem = items.find((item) => item.key === activeKey);

  return (
    <ACard
      styles={{ body: { padding: 0 } }}
      title={
        <PageHeaderTitle
          description={t('page.security.subtitle')}
          icon={<SafetyCertificateOutlined />}
          title={t('page.security.title')}
        />
      }
    >
      <Tabs
        className="settings-tabs"
        activeKey={activeKey}
        items={items}
        onChange={onChange}
      />
      <div className="settings-tabs-content">
        {activeItem?.component ? <activeItem.component /> : null}
      </div>
    </ACard>
  );
}
