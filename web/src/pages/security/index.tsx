import { Tabs } from "antd";

import Auth from "./modules/Auth";
import Role from "./modules/Role";

import "./index.scss";
import User from "./modules/User";
import { getProviderInfo } from "@/store/slice/server";

export function Component() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth();

  const permissions = {
    viewAuth: hasAuth("generic#security:authorization/search"),
    viewUser: hasAuth("generic#security:user/search"),
    viewRole: hasAuth("generic#security:role/search"),
  };

  const providerInfo = useAppSelector(getProviderInfo);

  const onChange = (key: string) => {
    setSearchParams({ tab: key });
  };

  const items: any[] = [];

  if (permissions.viewAuth && providerInfo?.managed) {
    items.push({
      component: Auth,
      key: "auth",
      label: t(`page.auth.title`),
    });
  }

  if (permissions.viewUser && !providerInfo?.managed) {
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

  const activeKey = useMemo(() => {
    return searchParams.get("tab") || items?.[0]?.key;
  }, []);

  const activeItem = useMemo(() => {
    return items.find((item) => item.key === activeKey);
  }, [activeKey]);

  return (
    <ACard styles={{ body: { padding: 0 } }}>
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
