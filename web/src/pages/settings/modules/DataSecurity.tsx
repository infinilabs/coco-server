import { DeleteOutlined, EyeOutlined, PlusOutlined, SafetyOutlined } from '@ant-design/icons';
import { Button, Card, Col, Divider, Input, Row, Select, Space, Switch, Table, Tooltip, Typography, message } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import '../index.scss';
import { fetchSettings, updateSettings } from '@/service/api/server';
import { fetchRoles } from '@/service/api/security';
import { useAuth } from '@/hooks/business/auth';

interface MaskingRule {
  id?: string;
  name?: string;
  pattern: string;
  replacement?: string;
  enabled: boolean;
}

interface FieldRestriction {
  role: string;
  exclude_fields: string[];
}

interface DataSecurityForm {
  masking: { enabled: boolean; rules: MaskingRule[] };
  field_access: { restrictions: FieldRestriction[] };
}

// RE2-safe presets (no lookaheads) so the server-side Go regex and the
// client-side preview agree
const PRESETS: MaskingRule[] = [
  { name: 'phone', pattern: '1[3-9]\\d{9}', replacement: '***PHONE***', enabled: true },
  { name: 'ID card', pattern: '\\d{17}[\\dXx]', replacement: '***ID***', enabled: true },
  { name: 'email', pattern: '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}', replacement: '***EMAIL***', enabled: true },
  { name: 'bank card', pattern: '\\b\\d{16,19}\\b', replacement: '***CARD***', enabled: true }
];

const FIELD_SUGGESTIONS = ['payload.*', 'raw_content', 'summary', 'ai_insights', 'content', 'document_chunk'];

const DataSecurity = () => {
  const { t } = useTranslation();
  const { hasAuth } = useAuth();

  const permissions = {
    update: hasAuth('coco#system/update')
  };

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [maskingEnabled, setMaskingEnabled] = useState(false);
  const [rules, setRules] = useState<MaskingRule[]>([]);
  const [restrictions, setRestrictions] = useState<FieldRestriction[]>([]);
  const [roleOptions, setRoleOptions] = useState<{ label: string; value: string }[]>([]);
  const [testInput, setTestInput] = useState('');

  useEffect(() => {
    setLoading(true);
    fetchSettings()
      .then((res: any) => {
        const cfg = res?.data?.data_security;
        setMaskingEnabled(!!cfg?.masking?.enabled);
        setRules(cfg?.masking?.rules ?? []);
        setRestrictions(cfg?.field_access?.restrictions ?? []);
      })
      .finally(() => setLoading(false));
    fetchRoles({ size: 100 }).then((res: any) => {
      const hits = res?.data?.hits?.hits ?? [];
      setRoleOptions(
        hits.map((hit: any) => ({ label: hit._source?.name ?? hit._id, value: hit._source?.name ?? hit._id }))
      );
    });
  }, []);

  const testOutput = useMemo(() => {
    if (!testInput) return '';
    let out = testInput;
    if (maskingEnabled) {
      for (const rule of rules) {
        if (!rule.enabled || !rule.pattern) continue;
        try {
          out = out.replace(new RegExp(rule.pattern, 'g'), rule.replacement ?? '');
        } catch {
          // broken pattern: server rejects it on save; preview skips it
        }
      }
    }
    return out;
  }, [testInput, rules, maskingEnabled]);

  const addRule = () => setRules([...rules, { pattern: '', replacement: '***', enabled: true }]);
  const addPresets = () => {
    const existing = new Set(rules.map((r) => r.pattern));
    setRules([...rules, ...PRESETS.filter((p) => !existing.has(p.pattern)).map((p) => ({ ...p }))]);
  };
  const addRestriction = () => setRestrictions([...restrictions, { role: '', exclude_fields: [] }]);

  const ruleColumns = [
    {
      title: t('page.settings.data_security.name'),
      width: 140,
      render: (_: any, row: MaskingRule, i: number) => (
        <Input
          value={row.name}
          disabled={!permissions.update}
          placeholder="rule name"
          onChange={(e) => setRules(rules.map((r, idx) => (idx === i ? { ...r, name: e.target.value } : r)))}
        />
      )
    },
    {
      title: t('page.settings.data_security.pattern'),
      render: (_: any, row: MaskingRule, i: number) => (
        <Input
          value={row.pattern}
          disabled={!permissions.update}
          className="font-mono text-12px"
          placeholder="1[3-9]\\d{9}"
          onChange={(e) => setRules(rules.map((r, idx) => (idx === i ? { ...r, pattern: e.target.value } : r)))}
        />
      )
    },
    {
      title: t('page.settings.data_security.replacement'),
      width: 150,
      render: (_: any, row: MaskingRule, i: number) => (
        <Input
          value={row.replacement}
          disabled={!permissions.update}
          className="font-mono text-12px"
          onChange={(e) => setRules(rules.map((r, idx) => (idx === i ? { ...r, replacement: e.target.value } : r)))}
        />
      )
    },
    {
      title: t('page.settings.data_security.enabled'),
      width: 80,
      render: (_: any, row: MaskingRule, i: number) => (
        <Switch
          size="small"
          checked={row.enabled}
          disabled={!permissions.update}
          onChange={(v) => setRules(rules.map((r, idx) => (idx === i ? { ...r, enabled: v } : r)))}
        />
      )
    },
    {
      title: '',
      width: 48,
      render: (_: any, __: MaskingRule, i: number) => (
        <Button
          type="text"
          size="small"
          danger
          icon={<DeleteOutlined />}
          disabled={!permissions.update}
          onClick={() => setRules(rules.filter((_, idx) => idx !== i))}
        />
      )
    }
  ];

  const handleSubmit = async () => {
    setSaving(true);
    try {
      const payload: DataSecurityForm = {
        masking: { enabled: maskingEnabled, rules },
        field_access: { restrictions: restrictions.filter((r) => r.role && r.exclude_fields.length > 0) }
      };
      const result = await updateSettings({ data_security: payload } as any);
      if (result?.data?.acknowledged) {
        window.$message?.success(t('common.updateSuccess'));
      }
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="p-24px flex flex-col gap-16px">
      <Card
        loading={loading}
        title={
          <Space size={8}>
            <SafetyOutlined />
            <span>{t('page.settings.data_security.maskingTitle')}</span>
          </Space>
        }
        extra={
          <Space>
            <span className="text-13px">{t('page.settings.data_security.maskingEnabled')}</span>
            <Switch checked={maskingEnabled} disabled={!permissions.update} onChange={setMaskingEnabled} />
          </Space>
        }
      >
        <Typography.Paragraph type="secondary" className="!mb-12px">
          {t('page.settings.data_security.maskingDesc')}
        </Typography.Paragraph>
        <Table
          rowKey={(_, i) => String(i)}
          size="small"
          pagination={false}
          columns={ruleColumns}
          dataSource={rules}
          locale={{ emptyText: t('page.settings.data_security.noRules') }}
        />
        <Space className="mt-12px">
          <Button size="small" icon={<PlusOutlined />} disabled={!permissions.update} onClick={addRule}>
            {t('page.settings.data_security.addRule')}
          </Button>
          <Button size="small" icon={<PlusOutlined />} disabled={!permissions.update} onClick={addPresets}>
            {t('page.settings.data_security.addPresets')}
          </Button>
        </Space>
        <Divider className="!my-16px">
          <Space size={4}>
            <EyeOutlined />
            <Typography.Text type="secondary" className="text-12px">
              {t('page.settings.data_security.tryIt')}
            </Typography.Text>
          </Space>
        </Divider>
        <Row gutter={12}>
          <Col span={12}>
            <Input.TextArea
              rows={3}
              value={testInput}
              onChange={(e) => setTestInput(e.target.value)}
              placeholder={t('page.settings.data_security.tryPlaceholder')}
            />
          </Col>
          <Col span={12}>
            <Input.TextArea rows={3} readOnly value={testOutput} placeholder={t('page.settings.data_security.tryOutput')} />
          </Col>
        </Row>
      </Card>

      <Card loading={loading} title={t('page.settings.data_security.fieldTitle')}>
        <Typography.Paragraph type="secondary" className="!mb-12px">
          {t('page.settings.data_security.fieldDesc')}
        </Typography.Paragraph>
        {restrictions.map((r, i) => (
          <Row gutter={12} key={i} className="mb-8px">
            <Col span={7}>
              <Select
                showSearch
                allowClear
                style={{ width: '100%' }}
                value={r.role || undefined}
                disabled={!permissions.update}
                placeholder={t('page.settings.data_security.rolePlaceholder')}
                options={roleOptions}
                onChange={(v) => setRestrictions(restrictions.map((x, idx) => (idx === i ? { ...x, role: v ?? '' } : x)))}
              />
            </Col>
            <Col span={15}>
              <Select
                mode="tags"
                allowClear
                style={{ width: '100%' }}
                value={r.exclude_fields}
                disabled={!permissions.update}
                tokenSeparators={[',']}
                placeholder="payload.*, raw_content"
                options={FIELD_SUGGESTIONS.map((f) => ({ label: f, value: f }))}
                onChange={(v) => setRestrictions(restrictions.map((x, idx) => (idx === i ? { ...x, exclude_fields: v } : x)))}
              />
            </Col>
            <Col span={2}>
              <Button
                type="text"
                danger
                icon={<DeleteOutlined />}
                disabled={!permissions.update}
                onClick={() => setRestrictions(restrictions.filter((_, idx) => idx !== i))}
              />
            </Col>
          </Row>
        ))}
        <Button size="small" icon={<PlusOutlined />} disabled={!permissions.update} onClick={addRestriction}>
          {t('page.settings.data_security.addRestriction')}
        </Button>
      </Card>

      {permissions.update ? (
        <div className="flex justify-end">
          <Button type="primary" loading={saving} onClick={handleSubmit}>
            {t('common.save')}
          </Button>
        </div>
      ) : null}
    </div>
  );
};

export default DataSecurity;
