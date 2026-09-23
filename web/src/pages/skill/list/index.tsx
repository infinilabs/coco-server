import { ExclamationCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { BookOpen, CheckCircle2, Download, ListChecks, Pencil, Plug, Search as SearchIcon, ShieldCheck, Trash } from 'lucide-react';
import { Button, Card, Drawer, Empty, Form, Input, Popconfirm, Select, Space, Switch, Tag, message } from 'antd';
import { useState } from 'react';

import useQueryParams from '@/hooks/common/queryParams';
import { createSkill, deleteSkill, exportMCPJSON, exportSkillsMD, searchSkill, updateSkill } from '@/service/api/skill';
import { formatESSearchResult } from '@/service/request/es';

type Skill = {
  id: string;
  name: string;
  title: string;
  description?: string;
  category?: string;
  instructions?: string;
  enabled: boolean;
  builtin: boolean;
  icon?: string;
  sort_order?: number;
};

// lucide icons referenced by the seed skills' icon field
const SKILL_ICONS: Record<string, React.ReactNode> = {
  Search: <SearchIcon size={18} />,
  ShieldCheck: <ShieldCheck size={18} />,
  BookOpen: <BookOpen size={18} />,
  Plug: <Plug size={18} />,
  ListChecks: <ListChecks size={18} />
};

const SKILL_CATEGORIES = ['retrieval', 'knowledge', 'onboarding', 'guardrails', 'productivity'];

function iconFor(skill: Skill) {
  return SKILL_ICONS[skill.icon ?? ''] ?? <CheckCircle2 size={18} />;
}

export function Component() {
  const [queryParams, setQueryParams] = useQueryParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth();

  const permissions = {
    create: hasAuth('coco#skill/create'),
    update: hasAuth('coco#skill/update'),
    delete: hasAuth('coco#skill/delete')
  };

  const [data, setData] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState<string>();
  const [editing, setEditing] = useState<Skill | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [form] = Form.useForm();

  const fetchData = async (params: any) => {
    setLoading(true);
    try {
      const res = await searchSkill(params);
      if (res?.data) {
        const result = formatESSearchResult(res.data);
        setData(result.data ?? []);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData(queryParams);
  }, [queryParams]);

  const onSearch = (query: string) => {
    setQueryParams((old: any) => ({
      ...old,
      from: query === old.query ? old.from : 0,
      query,
      t: Date.now()
    }));
  };

  const onEnabledChange = (skill: Skill, value: boolean) => {
    updateSkill(skill.id, { enabled: value }).then(() => {
      message.success(t('page.skill.toast.updated'));
      setQueryParams((old: any) => ({ ...old, t: Date.now() }));
    });
  };

  const onDelete = (skill: Skill) => {
    deleteSkill(skill.id).then(() => {
      message.success(t('common.deleteSuccess'));
      setQueryParams((old: any) => ({ ...old, t: Date.now() }));
    });
  };

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setDrawerOpen(true);
  };

  const openEdit = (skill: Skill) => {
    setEditing(skill);
    form.setFieldsValue(skill);
    setDrawerOpen(true);
  };

  const onSave = async () => {
    const values = await form.validateFields();
    if (editing) {
      await updateSkill(editing.id, values);
    } else {
      await createSkill(values);
    }
    message.success(t('page.skill.toast.updated'));
    setDrawerOpen(false);
    setQueryParams((old: any) => ({ ...old, t: Date.now() }));
  };

  const onDownloadSkillsMD = async () => {
    const res: any = await exportSkillsMD();
    const blob = new Blob([typeof res.data === 'string' ? res.data : JSON.stringify(res.data)], {
      type: 'text/markdown;charset=utf-8'
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'SKILLS.md';
    a.click();
    URL.revokeObjectURL(url);
  };

  const onCopyMCPJSON = async () => {
    const res: any = await exportMCPJSON();
    const text = JSON.stringify(res.data, null, 2);
    await navigator.clipboard.writeText(text);
    message.success(t('page.skill.export.copied'));
  };

  return (
    <div className="min-h-500px h-full p-16px">
      <Card
        bordered={false}
        className="card-wrapper"
        loading={loading && data.length === 0}
      >
        <div className="mb-16px flex flex-wrap items-center justify-between gap-12px">
          <div>
            <div className="text-18px font-500">{t('page.skill.title')}</div>
            <div className="text-13px color-[var(--ant-color-text-tertiary)]">{t('page.skill.subtitle')}</div>
          </div>
          <Space wrap>
            <Input
              allowClear
              className="w-220px"
              placeholder={t('page.skill.search')}
              prefix={<SearchIcon size={14} />}
              value={keyword}
              onChange={e => setKeyword(e.target.value)}
              onPressEnter={e => onSearch((e.target as HTMLInputElement).value)}
            />
            <Button
              icon={<Download size={14} />}
              onClick={onDownloadSkillsMD}
            >
              SKILLS.md
            </Button>
            <Button
              onClick={onCopyMCPJSON}
            >
              mcp.json
            </Button>
            {permissions.create && (
              <Button
                icon={<PlusOutlined />}
                type="primary"
                onClick={openCreate}
              >
                {t('common.add')}
              </Button>
            )}
          </Space>
        </div>

        {data.length === 0 && !loading ? (
          <Empty description={t('page.skill.empty')} />
        ) : (
          <div className="grid grid-cols-1 gap-12px sm:grid-cols-2 xl:grid-cols-3">
            {data.map(skill => (
              <Card
                key={skill.id}
                className="skill-card"
                hoverable
                size="small"
                styles={{ body: { padding: '14px 16px' } }}
              >
                <div className="flex items-start justify-between gap-8px">
                  <div className="flex flex-1 items-center gap-8px overflow-hidden">
                    <span className="flex-shrink-0 text-[var(--ant-color-primary)]">{iconFor(skill)}</span>
                    <span className="truncate text-15px font-500">{skill.title}</span>
                    {skill.builtin && (
                      <Tag
                        bordered={false}
                        color="blue"
                      >
                        {t('page.skill.builtin')}
                      </Tag>
                    )}
                  </div>
                  <Switch
                    checked={skill.enabled}
                    disabled={!permissions.update}
                    size="small"
                    onChange={v => onEnabledChange(skill, v)}
                  />
                </div>
                {skill.description && (
                  <div
                    className="mt-4px text-13px color-[var(--ant-color-text-secondary)]"
                    style={{ minHeight: '20px', display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden' }}
                  >
                    {skill.description}
                  </div>
                )}
                <div className="mt-8px flex items-center justify-between">
                  <Tag
                    bordered={false}
                    className="m-0"
                  >
                    {skill.category || 'general'}
                  </Tag>
                  <Space size={4}>
                    {permissions.update && (
                      <Button
                        size="small"
                        type="text"
                        onClick={() => openEdit(skill)}
                      >
                        <Pencil size={13} />
                      </Button>
                    )}
                    {permissions.delete && !skill.builtin && (
                      <Popconfirm
                        icon={<ExclamationCircleOutlined />}
                        title={t('page.skill.delete.confirm')}
                        onConfirm={() => onDelete(skill)}
                      >
                        <Button
                          danger
                          size="small"
                          type="text"
                        >
                          <Trash size={13} />
                        </Button>
                      </Popconfirm>
                    )}
                  </Space>
                </div>
              </Card>
            ))}
          </div>
        )}
      </Card>

      <Drawer
        destroyOnClose
        title={editing ? t('page.skill.edit') : t('page.skill.create')}
        width={520}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        extra={
          <Button
            type="primary"
            onClick={onSave}
          >
            {t('common.save')}
          </Button>
        }
      >
        <Form
          form={form}
          layout="vertical"
        >
          <Form.Item
            label={t('page.skill.labels.title')}
            name="title"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            label={t('page.skill.labels.name')}
            name="name"
            extra={editing?.builtin ? t('page.skill.name.readonly') : undefined}
          >
            <Input disabled={!!editing?.builtin} />
          </Form.Item>
          <Form.Item
            label={t('page.skill.labels.description')}
            name="description"
          >
            <Input.TextArea
              autoSize
              rows={1}
            />
          </Form.Item>
          <Form.Item
            label={t('page.skill.labels.category')}
            name="category"
          >
            <Select
              allowClear
              options={SKILL_CATEGORIES.map(c => ({ label: c, value: c }))}
            />
          </Form.Item>
          <Form.Item
            label={t('page.skill.labels.instructions')}
            name="instructions"
            rules={[{ required: true }]}
            extra={t('page.skill.instructions.hint')}
          >
            <Input.TextArea
              autoSize
              rows={10}
            />
          </Form.Item>
          <Form.Item
            label={t('page.skill.labels.enabled')}
            name="enabled"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
        </Form>
      </Drawer>
    </div>
  );
}
