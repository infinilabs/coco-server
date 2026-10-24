import { DeleteOutlined, PlusOutlined, SaveOutlined, ClusterOutlined } from '@ant-design/icons';
import PageHeaderTitle from '@/components/stateless/common/PageHeaderTitle';
import {
  AutoComplete,
  Button,
  Card,
  Col,
  Collapse,
  Empty,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Spin,
  Switch,
  Tag,
  Tooltip,
  message
} from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  getOntologySchema,
  putOntologySchema,
  searchWikiEntityPage,
  searchWikiKbs,
  type OntologyCompileRules,
  type OntologyEntityTypeDef,
  type OntologyPageSection,
  type OntologyPropertyDef,
  type OntologyRelationDef
} from '@/service/api';

const PROP_TYPES = ['string', 'text', 'number', 'boolean', 'date', 'url', 'enum'];

/** Local-only stable node ids: index-based keys make React reuse mismatched
 *  inputs after a mid-list delete (contents shift down one row). */
type TProp = OntologyPropertyDef & { _id: string };
type TRel = OntologyRelationDef & { _id: string };
type TSec = OntologyPageSection & { _id: string };
interface TType extends Omit<OntologyEntityTypeDef, 'properties' | 'relations' | 'page_sections'> {
  _id: string;
  properties?: TProp[];
  relations?: TRel[];
  page_sections?: TSec[];
}
const genId = () =>
  typeof crypto !== 'undefined' && crypto.randomUUID ? crypto.randomUUID() : `id-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

const withIds = (src: OntologyEntityTypeDef[]): TType[] =>
  (src || []).map(tp => ({
    ...tp,
    _id: genId(),
    properties: (tp.properties || []).map(p => ({ ...p, _id: genId() })),
    relations: (tp.relations || []).map(r => ({ ...r, _id: genId() })),
    page_sections: (tp.page_sections || []).map(s => ({ ...s, _id: genId() }))
  }));

/** strip local ids before persisting */
const stripIds = (tp: TType): OntologyEntityTypeDef => {
  const { _id, ...rest } = tp;
  return {
    ...rest,
    properties: (tp.properties || []).map(({ _id: a, ...p }) => p),
    relations: (tp.relations || []).map(({ _id: a, ...r }) => r),
    page_sections: (tp.page_sections || []).map(({ _id: a, ...s }) => s)
  };
};

/** per-type validation flags, keyed by node id (computed live, surfaced after first save attempt) */
interface TypeErrors {
  name: boolean;
  props: Record<string, boolean>;
  rels: Record<string, boolean>;
}

/**
 * Ontology vocabulary editor (phase O1): the declared entity types, their
 * typed properties and the relation vocabulary that entity writes validate
 * against. Scope is the tenant default or one knowledge base's override.
 */
export function Component() {
  const { t } = useTranslation();
  const [types, setTypes] = useState<TType[]>([]);
  const [rules, setRules] = useState<OntologyCompileRules>({});
  const [kbs, setKbs] = useState<Api.Wiki.Kb[]>([]);
  const [kbId, setKbId] = useState<string>('');
  const nav = useNavigate();
  const [loading, setLoading] = useState(true);
  const [loadFailed, setLoadFailed] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveAttempted, setSaveAttempted] = useState(false);
  /** serialized snapshot of the last loaded/saved state — anything else is dirty */
  const [baseline, setBaseline] = useState('{"types":[],"rules":{}}');
  const [openIds, setOpenIds] = useState<string[]>([]);
  const [delTarget, setDelTarget] = useState<TType | null>(null);
  const [delCount, setDelCount] = useState<number | null>(null);
  const [delCounting, setDelCounting] = useState(false);

  const load = (scopeKb?: string) => {
    setLoading(true);
    setSaveAttempted(false);
    getOntologySchema(scopeKb || undefined)
      .then(res => {
        const nextTypes = withIds((res as any)?.entity_types || []);
        const nextRules = ((res as any)?.rules || {}) as OntologyCompileRules;
        setTypes(nextTypes);
        setRules(nextRules);
        setBaseline(JSON.stringify({ types: nextTypes.map(stripIds), rules: nextRules }));
        // a handful of types reads fine expanded; a long list starts collapsed
        setOpenIds(nextTypes.length <= 5 ? nextTypes.map(tp => tp._id) : []);
        setLoadFailed(false);
        setLoading(false);
      })
      .catch(() => {
        setLoading(false);
        setLoadFailed(true);
        message.error(t('page.ontology.loadFailed'));
      });
  };

  useEffect(() => {
    load();
    searchWikiKbs().then(res => setKbs(((res as any)?.data || []) as Api.Wiki.Kb[]));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  /* ---------------- dirty tracking (#1) ---------------- */

  const snapshot = JSON.stringify({ types: types.map(stripIds), rules });
  const dirty = useMemo(() => snapshot !== baseline, [snapshot, baseline]);
  const dirtyRef = useRef(dirty);
  dirtyRef.current = dirty;

  // browser close/refresh guard; in-app scope switch is confirmed separately
  useEffect(() => {
    const h = (e: BeforeUnloadEvent) => {
      if (dirtyRef.current) {
        e.preventDefault();
        e.returnValue = '';
      }
    };
    window.addEventListener('beforeunload', h);
    return () => window.removeEventListener('beforeunload', h);
  }, []);

  const onScopeChange = (v: string) => {
    const doLoad = () => {
      setKbId(v);
      load(v || undefined);
    };
    if (dirtyRef.current) {
      Modal.confirm({
        title: t('page.ontology.scopeSwitchConfirm'),
        okButtonProps: { danger: true },
        onOk: doLoad
      });
    } else {
      doLoad();
    }
  };

  const save = () => {
    setSaveAttempted(true);
    const bad = types.filter(tp => {
      const e = vmap[tp._id];
      return e && (e.name || Object.values(e.props).some(Boolean) || Object.values(e.rels).some(Boolean));
    });
    if (bad.length) {
      setOpenIds(prev => Array.from(new Set([...prev, ...bad.map(tp => tp._id)])));
      message.error(t('page.ontology.validationError', { n: bad.length }));
      return;
    }
    setSaving(true);
    const clean = types.map(stripIds);
    putOntologySchema(clean, kbId || undefined, rules)
      .then(() => {
        setSaving(false);
        setBaseline(JSON.stringify({ types: clean, rules }));
        message.success(t('page.ontology.saved'));
      })
      .catch(() => {
        setSaving(false);
        message.error(t('page.ontology.saveFailed'));
      });
  };

  const discardChanges = () => load(kbId || undefined);

  /* ---------------- live validation (#3) ---------------- */

  const vmap = useMemo(() => {
    const map: Record<string, TypeErrors> = {};
    const nameCount = new Map<string, number>();
    types.forEach(tp => {
      const k = tp.name.trim().toLowerCase();
      if (k) nameCount.set(k, (nameCount.get(k) || 0) + 1);
    });
    types.forEach(tp => {
      const e: TypeErrors = { name: false, props: {}, rels: {} };
      const k = tp.name.trim().toLowerCase();
      // name doubles as the [[type:name]] identifier: non-empty, no spaces, unique
      if (!k || /\s/.test(tp.name.trim()) || (nameCount.get(k) || 0) > 1) e.name = true;
      const propSeen = new Map<string, number>();
      (tp.properties || []).forEach(p => {
        const pk = p.key.trim().toLowerCase();
        if (pk) propSeen.set(pk, (propSeen.get(pk) || 0) + 1);
      });
      (tp.properties || []).forEach(p => {
        const pk = p.key.trim().toLowerCase();
        if (!pk || (propSeen.get(pk) || 0) > 1) e.props[p._id] = true;
      });
      const relSeen = new Map<string, number>();
      (tp.relations || []).forEach(r => {
        const rk = r.name.trim().toLowerCase();
        if (rk) relSeen.set(rk, (relSeen.get(rk) || 0) + 1);
      });
      (tp.relations || []).forEach(r => {
        const rk = r.name.trim().toLowerCase();
        if (!rk || (relSeen.get(rk) || 0) > 1) e.rels[r._id] = true;
      });
      map[tp._id] = e;
    });
    return map;
  }, [types]);

  const nameErr = (tp: TType) => (saveAttempted && vmap[tp._id]?.name ? 'error' : undefined);

  /* ---------------- mutation helpers (#2 stable keys, #4 rename cascade) ---------------- */

  const mutateType = (idx: number, patch: Partial<TType>) => {
    setTypes(prev => prev.map((tp, i) => (i === idx ? { ...tp, ...patch } : tp)));
  };

  /** renaming a type cascades to every reference: relation target_type and
   *  page_sections keys of the form relations:<name> would dangle otherwise */
  const renameType = (idx: number, newName: string) => {
    setTypes(prev => {
      const old = prev[idx]?.name;
      const next = prev.map((tp, i) => (i === idx ? { ...tp, name: newName } : tp));
      if (!old || old === newName) return next;
      return next.map(tp => ({
        ...tp,
        relations: (tp.relations || []).map(r => (r.target_type === old ? { ...r, target_type: newName } : r)),
        page_sections: (tp.page_sections || []).map(s =>
          s.key === `relations:${old}` ? { ...s, key: `relations:${newName}` } : s
        )
      }));
    });
  };

  const addType = () => {
    const nt: TType = { _id: genId(), name: `type_${types.length + 1}`, properties: [], relations: [] };
    setTypes(prev => [...prev, nt]);
    setOpenIds(prev => [...prev, nt._id]);
  };

  const addProperty = (typeIdx: number) => {
    setTypes(prev =>
      prev.map((tp, i) =>
        i === typeIdx ? { ...tp, properties: [...(tp.properties || []), { _id: genId(), key: '', type: 'string' }] } : tp
      )
    );
  };

  const addRelation = (typeIdx: number) => {
    setTypes(prev =>
      prev.map((tp, i) =>
        i === typeIdx ? { ...tp, relations: [...(tp.relations || []), { _id: genId(), name: '', target_type: '*' }] } : tp
      )
    );
  };

  const addPageSection = (typeIdx: number) => {
    setTypes(prev =>
      prev.map((tp, i) =>
        i === typeIdx ? { ...tp, page_sections: [...(tp.page_sections || []), { _id: genId(), key: '', title: '' }] } : tp
      )
    );
  };

  /* ---------------- delete with entity usage count (#8) ---------------- */

  const askDelete = (tp: TType) => {
    setDelTarget(tp);
    setDelCount(null);
    setDelCounting(true);
    searchWikiEntityPage({ type: tp.name, kbId: kbId || undefined, size: 1 })
      .then(r => setDelCount(r.total ?? 0))
      .catch(() => setDelCount(null))
      .finally(() => setDelCounting(false));
  };

  const confirmDelete = () => {
    if (!delTarget) return;
    const id = delTarget._id;
    setTypes(prev => prev.filter(tp => tp._id !== id));
    setOpenIds(prev => prev.filter(k => k !== id));
    setDelTarget(null);
  };

  // key suggestions: the data-driven built-ins plus one entry per declared
  // relation of THIS type (relations:<name> renders that group as wikilinks)
  const sectionKeyOptions = (tp: TType) => {
    const base = [
      { value: 'sources', label: 'sources — 来源引用' },
      { value: 'aliases', label: 'aliases — 别名' },
      { value: 'properties', label: 'properties — 类型化属性' }
    ];
    const rels = (tp.relations || [])
      .filter(r => r.name)
      .map(r => ({ value: `relations:${r.name}`, label: `relations:${r.name} — ${r.label || r.name}` }));
    return [...base, ...rels];
  };

  const typeNames = ['*', ...types.map(tp => tp.name)];

  const renderTypePanel = (tp: TType, typeIdx: number) => (
    <>
      <div className='mb-12px flex flex-wrap items-center gap-8px'>
        <Input
          className='w-160px'
          onChange={e => renameType(typeIdx, e.target.value)}
          placeholder={t('page.ontology.typeName')}
          size='small'
          status={nameErr(tp)}
          value={tp.name}
        />
        <Input
          className='w-160px'
          onChange={e => mutateType(typeIdx, { label: e.target.value })}
          placeholder={t('page.ontology.typeLabel')}
          size='small'
          value={tp.label}
        />
        <Tooltip title={t('page.ontology.iconHint')}>
          <Input
            className='w-64px'
            onChange={e => mutateType(typeIdx, { icon: e.target.value })}
            placeholder='🙂'
            size='small'
            value={tp.icon}
          />
        </Tooltip>
        <Tag className='ml-auto' color='default'>
          {t('page.ontology.typeSummary', {
            p: tp.properties?.length || 0,
            r: tp.relations?.length || 0,
            s: tp.page_sections?.length || 0
          })}
        </Tag>
      </div>
      <div className='grid grid-cols-1 gap-12px xl:!grid-cols-2'>
        {/* properties */}
        <div>
          <div className='mb-2 flex items-center justify-between'>
            <span className='font-medium'>{t('page.ontology.properties')}</span>
            <Button icon={<PlusOutlined />} onClick={() => addProperty(typeIdx)} size='small' type='text' />
          </div>
          {(tp.properties || []).map((prop: TProp) => (
            <div className='mb-2 flex items-center gap-4px' key={prop._id}>
              <Input
                className='w-110px'
                onChange={e => {
                  const next = (tp.properties || []).map(p => (p._id === prop._id ? { ...p, key: e.target.value } : p));
                  mutateType(typeIdx, { properties: next });
                }}
                placeholder={t('page.ontology.propKey')}
                size='small'
                status={saveAttempted && vmap[tp._id]?.props[prop._id] ? 'error' : undefined}
                value={prop.key}
              />
              <Input
                className='w-110px'
                onChange={e => {
                  const next = (tp.properties || []).map(p => (p._id === prop._id ? { ...p, label: e.target.value } : p));
                  mutateType(typeIdx, { properties: next });
                }}
                placeholder={t('page.ontology.propLabel')}
                size='small'
                value={prop.label}
              />
              <Select
                className='w-90px'
                onChange={v => {
                  const next = (tp.properties || []).map(p => (p._id === prop._id ? { ...p, type: v } : p));
                  mutateType(typeIdx, { properties: next });
                }}
                options={PROP_TYPES.map(v => ({ value: v }))}
                size='small'
                value={prop.type}
              />
              {prop.type === 'enum' && (
                <Tooltip title={t('page.ontology.enumHint')}>
                  <Input
                    className='w-140px'
                    onChange={e => {
                      const next = (tp.properties || []).map(p =>
                        p._id === prop._id ? { ...p, enum: e.target.value.split(',').map(s => s.trim()).filter(Boolean) } : p
                      );
                      mutateType(typeIdx, { properties: next });
                    }}
                    placeholder='a,b,c'
                    size='small'
                    value={(prop.enum || []).join(',')}
                  />
                </Tooltip>
              )}
              <Tooltip title={t('page.ontology.required')}>
                <Switch
                  checked={!!prop.required}
                  onChange={v => {
                    const next = (tp.properties || []).map(p => (p._id === prop._id ? { ...p, required: v } : p));
                    mutateType(typeIdx, { properties: next });
                  }}
                  size='small'
                />
              </Tooltip>
              <Button
                danger
                icon={<DeleteOutlined />}
                onClick={() => mutateType(typeIdx, { properties: (tp.properties || []).filter(p => p._id !== prop._id) })}
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
          {(tp.relations || []).map((rel: TRel) => (
            <div className='mb-2 flex items-center gap-4px' key={rel._id}>
              <Input
                className='w-110px'
                onChange={e => {
                  const next = (tp.relations || []).map(r => (r._id === rel._id ? { ...r, name: e.target.value } : r));
                  mutateType(typeIdx, { relations: next });
                }}
                placeholder={t('page.ontology.relName')}
                size='small'
                status={saveAttempted && vmap[tp._id]?.rels[rel._id] ? 'error' : undefined}
                value={rel.name}
              />
              <Select
                className='w-130px'
                onChange={v => {
                  const next = (tp.relations || []).map(r => (r._id === rel._id ? { ...r, target_type: v } : r));
                  mutateType(typeIdx, { relations: next });
                }}
                options={[{ value: '*', label: '* — 任意类型' }, ...types.filter(x => x.name).map(x => ({ value: x.name, label: x.name }))]}
                size='small'
                showSearch
                value={rel.target_type}
              />
              <Select
                allowClear
                className='w-90px'
                onChange={v => {
                  const next = (tp.relations || []).map(r => (r._id === rel._id ? { ...r, cardinality: v } : r));
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
                  const next = (tp.relations || []).map(r => (r._id === rel._id ? { ...r, inverse: e.target.value } : r));
                  mutateType(typeIdx, { relations: next });
                }}
                placeholder={t('page.ontology.inverse')}
                size='small'
                value={rel.inverse}
              />
              <Button
                danger
                icon={<DeleteOutlined />}
                onClick={() => mutateType(typeIdx, { relations: (tp.relations || []).filter(r => r._id !== rel._id) })}
                size='small'
                type='text'
              />
            </div>
          ))}
        </div>
      </div>
      {/* page layout: the section list a generated page follows */}
      <div className='mt-12px'>
        <div className='mb-2 flex items-center justify-between'>
          <span className='font-medium'>{t('page.ontology.pageSections')}</span>
          <Button icon={<PlusOutlined />} onClick={() => addPageSection(typeIdx)} size='small' type='text' />
        </div>
        <div className='mb-4px text-gray-500'>{t('page.ontology.pageSectionsHint')}</div>
        {(tp.page_sections || []).map((sec: TSec) => (
          <div className='mb-2 flex items-center gap-4px' key={sec._id}>
            <AutoComplete
              className='w-220px'
              onChange={v => {
                const next = (tp.page_sections || []).map(s => (s._id === sec._id ? { ...s, key: v } : s));
                mutateType(typeIdx, { page_sections: next });
              }}
              options={sectionKeyOptions(tp)}
              placeholder={t('page.ontology.sectionKey')}
              size='small'
              value={sec.key}
            />
            <Input
              className='w-180px'
              onChange={e => {
                const next = (tp.page_sections || []).map(s => (s._id === sec._id ? { ...s, title: e.target.value } : s));
                mutateType(typeIdx, { page_sections: next });
              }}
              placeholder={t('page.ontology.sectionTitle')}
              size='small'
              value={sec.title}
            />
            <Button
              danger
              icon={<DeleteOutlined />}
              onClick={() => mutateType(typeIdx, { page_sections: (tp.page_sections || []).filter(s => s._id !== sec._id) })}
              size='small'
              type='text'
            />
          </div>
        ))}
      </div>
    </>
  );

  return (
    <div className='min-h-500px'>
      <Card
        bordered={false}
        className='card-wrapper'
        title={
          <PageHeaderTitle
            description={t('page.ontology.subtitle')}
            icon={<ClusterOutlined />}
            title={t('page.ontology.title')}
          />
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
            {kbId && <Button onClick={() => nav(`/wiki/kb/${kbId}?tab=entities`)}>{t('page.ontology.viewEntities')}</Button>}
            <Button icon={<PlusOutlined />} onClick={addType}>
              {t('page.ontology.addType')}
            </Button>
          </Space>
        }
      >
        {/* knowledge compile rules: low-frequency config, collapsed by default (#9) */}
        <Collapse
          className='mb-16px'
          items={[
            {
              key: 'rules',
              label: <span className='font-medium'>{t('page.ontology.rules.title')}</span>,
              children: (
                <>
                  <div className='mb-8px text-gray-500'>{t('page.ontology.rules.desc')}</div>
                  <Row gutter={[16, 8]}>
                    <Col span={6}>
                      <div className='mb-4px'>{t('page.ontology.rules.conceptMinSources')}</div>
                      <InputNumber
                        min={1}
                        max={10}
                        style={{ width: '100%' }}
                        value={rules.concept_min_sources ?? 2}
                        onChange={v => setRules({ ...rules, concept_min_sources: v ?? 2 })}
                      />
                    </Col>
                    <Col span={6}>
                      <div className='mb-4px'>{t('page.ontology.rules.conflictStrategy')}</div>
                      <Select
                        style={{ width: '100%' }}
                        value={rules.conflict_strategy || 'mark'}
                        onChange={v => setRules({ ...rules, conflict_strategy: v })}
                        options={[
                          { value: 'mark', label: t('page.ontology.rules.strategyMark') },
                          { value: 'isolate', label: t('page.ontology.rules.strategyIsolate') }
                        ]}
                      />
                    </Col>
                    <Col span={12}>
                      <div className='mb-4px'>{t('page.ontology.rules.sourcePriority')}</div>
                      <Select
                        mode='tags'
                        allowClear
                        style={{ width: '100%' }}
                        open={false}
                        tokenSeparators={[',']}
                        placeholder={t('page.ontology.rules.sourcePriorityPlaceholder')}
                        value={rules.source_priority || []}
                        onChange={v => setRules({ ...rules, source_priority: v })}
                      />
                    </Col>
                  </Row>
                </>
              )
            }
          ]}
        />

        <Spin spinning={loading}>
          {loadFailed && (
            <Empty description={t('page.ontology.loadFailed')} image={Empty.PRESENTED_IMAGE_SIMPLE}>
              <Button onClick={() => load(kbId || undefined)} type='primary'>
                {t('page.ontology.retry')}
              </Button>
            </Empty>
          )}
          {!loadFailed && !loading && types.length === 0 && (
            <Empty description={t('page.ontology.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
          )}

          {/* one collapsible panel per type, header carries an at-a-glance summary (#7) */}
          {types.length > 0 && (
            <Collapse
              activeKey={openIds}
              className='mb-8px'
              items={types.map((tp, typeIdx) => ({
                key: tp._id,
                label: (
                  <div className='flex items-center gap-8px'>
                    <span>{tp.icon}</span>
                    <span className={saveAttempted && vmap[tp._id]?.name ? 'color-[var(--ant-color-error)] font-medium' : 'font-medium'}>
                      {tp.name || t('page.ontology.typeName')}
                    </span>
                    {tp.label && <span className='text-gray-500'>{tp.label}</span>}
                  </div>
                ),
                extra: (
                  // stopPropagation so the delete button doesn't toggle the panel
                  <span onClick={e => e.stopPropagation()}>
                    <Tooltip title={t('page.ontology.removeType')}>
                      <Button danger icon={<DeleteOutlined />} onClick={() => askDelete(tp)} size='small' type='text' />
                    </Tooltip>
                  </span>
                ),
                children: renderTypePanel(tp, typeIdx)
              }))}
              onChange={keys => setOpenIds(Array.isArray(keys) ? keys : [keys])}
            />
          )}
        </Spin>
      </Card>

      {/* sticky save bar: reachable wherever the user is on a long page (#6) */}
      <div
        className='sticky bottom-0 z-20 -mx-16px mt-16px flex items-center gap-12px px-16px py-10px'
        style={{
          background: 'var(--ant-color-bg-container)',
          borderTop: '1px solid var(--ant-color-split)',
          marginBottom: '-16px'
        }}
      >
        <span style={{ color: dirty ? 'var(--ant-color-warning)' : 'var(--ant-color-text-quaternary)' }}>
          {dirty ? t('page.ontology.unsavedTitle') : t('page.ontology.allSaved')}
        </span>
        <div className='ml-auto flex gap-8px'>
          <Button disabled={!dirty} onClick={discardChanges}>
            {t('page.ontology.discard')}
          </Button>
          <Button disabled={!dirty} icon={<SaveOutlined />} loading={saving} onClick={save} type='primary'>
            {t('page.ontology.save')}
          </Button>
        </div>
      </div>

      {/* delete confirm carries the entity usage count (#8) */}
      <Modal
        okButtonProps={{ danger: true }}
        okText={t('page.ontology.removeType')}
        onCancel={() => setDelTarget(null)}
        onOk={confirmDelete}
        open={!!delTarget}
        title={t('page.ontology.removeTypeConfirm', { name: delTarget?.name || '' })}
      >
        <div className='mt-8px'>
          {delCounting ? (
            <span className='text-gray-500'>{t('page.ontology.entityCounting')}</span>
          ) : delCount !== null && delCount > 0 ? (
            <span style={{ color: 'var(--ant-color-warning)' }}>{t('page.ontology.entityCount', { n: delCount })}</span>
          ) : (
            <span className='text-gray-500'>{t('page.ontology.entityNone')}</span>
          )}
        </div>
      </Modal>
    </div>
  );
}
