import { Button, Modal, Select, Space, Tag } from 'antd';
import { PlusOutlined, CloseOutlined } from '@ant-design/icons';
import { useRequest, useLoading } from '@sa/hooks';
import { fetchGetUserList, fetchGetAllRoles } from '@/service/api/system-manage';

type SubjectType = 'user' | 'team';

interface AddAuthItem {
  subjectType: SubjectType;
  subject?: { id: string; name: string };
  roles: Array<{ value: string; label: string }>;
}

interface AddAuthProps {
  open: boolean;
  onCancel: () => void;
  onSubmit: (items: AddAuthItem[]) => void;
  teamOptions?: Array<{ value: string; label: string }>;
}

export default function AddAuth(props: AddAuthProps) {
  const { open, onCancel, onSubmit, teamOptions = [{ value: 'app_user', label: 'APP 用户' }] } = props;
  const { t } = useTranslation();
  const { startLoading, endLoading, loading } = useLoading();

  const { data: rolesRes, run: fetchRoles } = useRequest(fetchGetAllRoles, { manual: true });
  const { data: usersRes, run: fetchUsers } = useRequest(fetchGetUserList, { manual: true });

  const [items, setItems] = useState<AddAuthItem[]>([
    { subjectType: 'user', subject: undefined, roles: [] }
  ]);

  useEffect(() => {
    fetchRoles();
    fetchUsers({ current: 1, size: 100 });
  }, []);

  const roleOptions = useMemo(() => {
    const list = Array.isArray(rolesRes?.data) ? rolesRes?.data : rolesRes;
    return (list || []).map((r: any) => ({
      value: r.roleCode || r.id || r.value,
      label: r.roleName || r.label
    }));
  }, [JSON.stringify(rolesRes)]);

  const userOptions = useMemo(() => {
    const records = usersRes?.data?.records || usersRes?.records || [];
    return records.map((u: any) => ({
      value: u.id || u.userName,
      label: u.nickName || u.userName
    }));
  }, [JSON.stringify(usersRes)]);

  const setItem = (index: number, patch: Partial<AddAuthItem>) => {
    setItems((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], ...patch };
      return next;
    });
  };

  const addRow = () => {
    setItems((prev) => prev.concat([{ subjectType: 'user', subject: undefined, roles: [] }]));
  };

  const removeRow = (index: number) => {
    setItems((prev) => prev.filter((_, i) => i !== index));
  };

  const handleSubmit = async () => {
    startLoading();
    try {
      onSubmit(items);
    } finally {
      endLoading();
    }
  };

  return (
    <Modal
      title="添加授权"
      open={open}
      onCancel={onCancel}
      footer={
        <Space>
          <Button onClick={onCancel}>{t('common.cancel')}</Button>
          <Button type="primary" onClick={handleSubmit} loading={loading}>
            {t('common.save')}
          </Button>
        </Space>
      }
      width={720}
    >
      <Space direction="vertical" className="w-full" size={16}>
        {items.map((item, index) => (
          <div key={index} className="flex items-center gap-12">
            <Select
              className="w-120px"
              value={item.subjectType}
              options={[
                { value: 'user', label: '人员' },
                { value: 'team', label: '团队' }
              ]}
              onChange={(v: SubjectType) => {
                setItem(index, { subjectType: v, subject: undefined });
              }}
            />
            <Select
              className="flex-1"
              placeholder={item.subjectType === 'user' ? '请选择人员' : '请选择团队'}
              value={item.subject?.value || item.subject?.id}
              options={item.subjectType === 'user' ? userOptions : teamOptions}
              showSearch
              onChange={(value: string, option: any) => {
                const name = option?.label || option?.children;
                setItem(index, { subject: { id: value, name } });
              }}
            />
            <span className="mx-8">Role</span>
            <Select
              className="flex-1"
              mode="multiple"
              placeholder="请选择角色"
              value={item.roles.map((r) => r.value)}
              options={roleOptions}
              maxTagCount="responsive"
              tagRender={(tag) => <Tag>{tag.label}</Tag>}
              onChange={(values: string[], options: any[]) => {
                const roles = (options || []).map((o: any) => ({ value: o.value, label: o.label }));
                setItem(index, { roles });
              }}
            />
            <Button
              type="text"
              danger
              icon={<CloseOutlined />}
              onClick={() => removeRow(index)}
            />
          </div>
        ))}
        <Button type="dashed" icon={<PlusOutlined />} onClick={addRow}>
          {t('common.add')}
        </Button>
      </Space>
    </Modal>
  );
}