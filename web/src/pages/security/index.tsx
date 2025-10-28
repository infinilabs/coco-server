import { Tabs } from "antd";
import type { TabsProps } from "antd";

import Auth from "./modules/Auth";
import Role from "./modules/Role";

import "./index.scss";
import User from "./modules/User";

export function Component() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth();

  const permissions = {
    viewRole: hasAuth("coco:role/view"),
    viewAuth: true,
    viewUser: true,
  };

  const onChange = (key: string) => {
    setSearchParams({ tab: key });
  };

  const items: TabsProps["items"] = [];

  if (permissions.viewAuth) {
    items.push({
      children: <Auth />,
      key: "auth",
      label: t(`page.auth.title`),
    });
  }

  if (permissions.viewUser) {
    items.push({
      children: <User />,
      key: "user",
      label: t(`page.user.title`),
    });
  }

  if (permissions.viewRole) {
    items.push({
      children: <Role />,
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
        {activeItem?.children ? activeItem.children : null}
      </div>
    </ACard>
  );
}
