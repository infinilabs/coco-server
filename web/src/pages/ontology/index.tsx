import { DeleteOutlined, PlusOutlined, SaveOutlined, ClusterOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Input, Modal, Select, Space, Switch, Tag, Tooltip, message } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  getOntologySchema,
  putOntologySchema,
  searchWikiKbs,
  type OntologyEntityTypeDef,
  type OntologyPropertyDef,
  type OntologyRelationDef
} from '@/service/api';

const PROP_TYPES = ['string', 'text', 'number', 'boolean', 'date', 'url', 'enum'];

/**
 * Ontology vocabulary editor (phase O1): the declared entity types, their
 * typed properties and the relation vocabulary that entity writes validate
 * against. Scope is the tenant default or one knowledge base's override.
 */
export function Component() {
  const { t } = useTranslation();
  const [types, setTypes] = useState<OntologyEntityTypeDef[]>([]);
  const [kbs, setKbs] = useState<Api.Wiki.Kb[]>([]);
  const [kbId, setKbId] = useState<string>('');
  const nav = useNavigate();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const load = (scopeKb?: string) => {
    setLoading(true);
    getOntologySchema(scopeKb || undefined).then(res => {
      setTypes(((res as any)?.entity_types || []) as OntologyEntityTypeDef[]);
      setLoading(false);
    });
  };

  useEffect(() => {
    load();
    searchWikiKbs().then(res => setKbs(((res as any)?.data || []) as Api.Wiki.Kb[]));
  }, []);

  const onScopeChange = (v: string) => {
    setKbId(v);
    load(v || undefined);
  };

  const save = () => {
    setSaving(true);
    putOntologySchema(types, kbId || undefined)
      .then(res => {
        setSaving(false);
        if ((res as any)?._id || (res as any)?.result === 'updated') {
          message.success(t('page.ontology.saved'));
        }
      })
      .catch(() => setSaving(false));
  };

  /* ---------------- mutation helpers ---------------- */

  const mutateType = (idx: number, patch: Partial<OntologyEntityTypeDef>) => {
    setTypes(prev => prev.map((tp, i) => (i === idx ? { ...tp, ...patch } : tp)));
  };

  const addType = () => {
    setTypes(prev => [...prev, { name: `type_${prev.length + 1}`, properties: [], relations: [] }]);
  };

  const addProperty = (typeIdx: number) => {
    setTypes(prev =>
      prev.map((tp, i) =>
        i === typeIdx ? { ...tp, properties: [...(tp.properties || []), { key: '', type: 'string' }] } : tp
      )
    );
  };

  const addRelation = (typeIdx: number) => {
    setTypes(prev =>
      prev.map((tp, i) =>
        i === typeIdx ? { ...tp, relations: [...(tp.relations || []), { name: '', target_type: '*' }] } : tp
      )
    );
  };

  const typeNames = ['*', ...types.map(tp => tp.name)];

  return (
    <div className='min-h-500px'>
      <Card
        bordered={false}
        className='card-wrapper'
        title={
          <Space>
            <ClusterOutlined />
            <span>{t('page.ontology.title')}</span>
          </Space>
        }
        extra={
          <Space>
            <Select
              allowClear
              className='w-220px'
              onChange={v => onScopeChange(v || '')}
              options={[
                { value: '', label: t('page.ontology.tenantScope') },
                ...kbs.map(kb => ({ value: kb.id, label: `${kb.icon || '📚'} ${kb.name}` }))
              ]}
              placeholder={t('page.ontology.tenantScope')}
              value={kbId || undefined}
            />
            {kbId && (
              <Button onClick={() => nav(`/wiki/kb/${kbId}?tab=entities`)}>{t('page.ontology.viewEntities')}</Button>
            )}
            <Button icon={<PlusOutlined />} onClick={addType}>
              {t('page.ontology.addType')}
            </Button>
            <Button icon={<SaveOutlined />} loading={saving} onClick={save} type='primary'>
              {t('page.ontology.save')}
            </Button>
          </Space>
        }
      >
        <div className='mb-4 text-gray-500'>{t('page.ontology.subtitle')}</div>

        {!loading && types.length === 0 && (
          <Empty description={t('page.ontology.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}

        <div className='flex flex-col gap-16px'>
          {types.map((tp, typeIdx) => (
            <Card key={typeIdx} size='small' title={
              <Space>
                <Input
                  className='w-140px'
                  onChange={e => mutateType(typeIdx, { name: e.target.value })}
                  placeholder={t('page.ontology.typeName')}
                  size='small'
                  value={tp.name}
                />
                <Input
                  className='w-140px'
                  onChange={e => mutateType(typeIdx, { label: e.target.value })}
                  placeholder={t('page.ontology.typeLabel')}
                  size='small'
                  value={tp.label}
                />
                <Input
                  className='w-64px'
                  onChange={e => mutateType(typeIdx, { icon: e.target.value })}
                  size='small'
                  value={tp.icon}
                />
              </Space>
            } extra={
              <Tooltip title={t('page.ontology.removeType')}>
                <Button
                  danger
                  icon={<DeleteOutlined />}
                  onClick={() => Modal.confirm({
                    title: t('page.ontology.removeTypeConfirm', { name: tp.name }),
                    onOk: () => setTypes(prev => prev.filter((_, i) => i !== typeIdx))
                  })}
                  size='small'
                  type='text'
                />
              </Tooltip>
            }>
              <div className='grid grid-cols-1 gap-12px xl:!grid-cols-2'>
                {/* properties */}
                <div>
                  <div className='mb-2 flex items-center justify-between'>
                    <span className='font-medium'>{t('page.ontology.properties')}</span>
                    <Button icon={<PlusOutlined />} onClick={() => addProperty(typeIdx)} size='small' type='text' />
                  </div>
                  {(tp.properties || []).map((prop: OntologyPropertyDef, propIdx) => (
                    <div className='mb-2 flex items-center gap-4px' key={propIdx}>
                      <Input
                        className='w-100px'
                        onChange={e => {
                          const next = [...(tp.properties || [])];
                          next[propIdx] = { ...prop, key: e.target.value };
                          mutateType(typeIdx, { properties: next });
                        }}
                        placeholder={t('page.ontology.propKey')}
                        size='small'
                        value={prop.key}
                      />
                      <Input
                        className='w-100px'
                        onChange={e => {
                          const next = [...(tp.properties || [])];
                          next[propIdx] = { ...prop, label: e.target.value };
                          mutateType(typeIdx, { properties: next });
                        }}
                        placeholder={t('page.ontology.propLabel')}
                        size='small'
                        value={prop.label}
                      />
                      <Select
                        className='w-90px'
                        onChange={v => {
                          const next = [...(tp.properties || [])];
                          next[propIdx] = { ...prop, type: v };
                          mutateType(typeIdx, { properties: next });
                        }}
                        options={PROP_TYPES.map(v => ({ value: v }))}
                        size='small'
                        value={prop.type}
                      />
                      {prop.type === 'enum' && (
                        <Input
                          className='w-140px'
                          onChange={e => {
                            const next = [...(tp.properties || [])];
                            next[propIdx] = { ...prop, enum: e.target.value.split(',').map(s => s.trim()).filter(Boolean) };
                            mutateType(typeIdx, { properties: next });
                          }}
                          placeholder={t('page.ontology.enumHint')}
                          size='small'
                          value={(prop.enum || []).join(',')}
                        />
                      )}
                      <Tooltip title={t('page.ontology.required')}>
                        <Switch
                          checked={!!prop.required}
                          onChange={v => {
                            const next = [...(tp.properties || [])];
                            next[propIdx] = { ...prop, required: v };
                            mutateType(typeIdx, { properties: next });
                          }}
                          size='small'
                        />
                      </Tooltip>
                      <Button
                        danger
                        icon={<DeleteOutlined />}
                        onClick={() => mutateType(typeIdx, {
                          properties: (tp.properties || []).filter((_, i) => i !== propIdx)
                        })}
                        size='small'
                        type='text'
                      />
                    </div>
                  ))}
                </div>

                {/* relations */}
                <div>
                  <div className='mb-2 flex items-center justify-between'>
                    <span className='font-medium'>{t('page.ontology.relations')}</span>
                    <Button icon={<PlusOutlined />} onClick={() => addRelation(typeIdx)} size='small' type='text' />
                  </div>
                  {(tp.relations || []).map((rel: OntologyRelationDef, relIdx) => (
                    <div className='mb-2 flex items-center gap-4px' key={relIdx}>
                      <Input
                        className='w-110px'
                        onChange={e => {
                          const next = [...(tp.relations || [])];
                          next[relIdx] = { ...rel, name: e.target.value };
                          mutateType(typeIdx, { relations: next });
                        }}
                        placeholder={t('page.ontology.relName')}
                        size='small'
                        value={rel.name}
                      />
                      <Select
                        className='w-120px'
                        onChange={v => {
                          const next = [...(tp.relations || [])];
                          next[relIdx] = { ...rel, target_type: v };
                          mutateType(typeIdx, { relations: next });
                        }}
                        options={typeNames.map(v => ({ value: v }))}
                        size='small'
                        value={rel.target_type}
                      />
                      <Select
                        allowClear
                        className='w-90px'
                        onChange={v => {
                          const next = [...(tp.relations || [])];
                          next[relIdx] = { ...rel, cardinality: v };
                          mutateType(typeIdx, { relations: next });
                        }}
                        options={[{ value: 'one' }, { value: 'many' }]}
                        placeholder={t('page.ontology.cardinality')}
                        size='small'
                        value={rel.cardinality}
                      />
                      <Input
                        className='w-110px'
                        onChange={e => {
                          const next = [...(tp.relations || [])];
                          next[relIdx] = { ...rel, inverse: e.target.value };
                          mutateType(typeIdx, { relations: next });
                        }}
                        placeholder={t('page.ontology.inverse')}
                        size='small'
                        value={rel.inverse}
                      />
                      <Button
                        danger
                        icon={<DeleteOutlined />}
                        onClick={() => mutateType(typeIdx, {
                          relations: (tp.relations || []).filter((_, i) => i !== relIdx)
                        })}
                        size='small'
                        type='text'
                      />
                    </div>
                  ))}
                </div>
              </div>
              {tp.properties?.length === 0 && tp.relations?.length === 0 && (
                <Tag>{t('page.ontology.bareType')}</Tag>
              )}
            </Card>
          ))}
        </div>
      </Card>
    </div>
  );
}
