import { ArrowRightOutlined, DeleteOutlined, PlusOutlined, SaveOutlined, ClusterOutlined } from '@ant-design/icons';
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
  Table,
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

/** color-coded type tags make the schema scannable at a glance */
const PROP_TYPE_COLOR: Record<string, string> = {
  string: 'blue',
  text: 'default',
  number: 'purple',
  boolean: 'geekblue',
  date: 'cyan',
  url: 'green',
  enum: 'orange'
};

/** notation-style cardinality labels (was bare "one"/"many") */
const CARD_LABEL: Record<string, string> = { one: '1..1', many: '0..n' };

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

  /** B: interface-style declaration preview — shows what the type "looks
   *  like" without parsing raw input rows */
  const renderPreview = (tp: TType) => {
    const props = tp.properties || [];
    const rels = tp.relations || [];
    if (!props.length && !rels.length) return null;
    return (
      <div
        className='mb-12px rounded-6px border border-solid px-12px py-8px'
        style={{ background: 'var(--ant-color-fill-quaternary)', borderColor: 'var(--ant-color-border-secondary)' }}
      >
        <div className='break-all font-mono text-12px leading-22px'>
          <span style={{ color: 'var(--ant-color-text-quaternary)' }}>interface</span>{' '}
          <span style={{ color: 'var(--ant-color-primary)' }}>{tp.name || '…'}</span>
          {' {'}
          {props.map(p => (
            <div className='pl-16px' key={p._id}>
              <span style={{ color: 'var(--ant-color-primary-text)' }}>{p.key || '…'}</span>
              {p.required ? <span style={{ color: 'var(--ant-color-error)' }}>*</span> : null}
              {': '}
              <span style={{ color: 'var(--ant-color-info-text)' }}>{p.type}</span>
              {p.type === 'enum' && (p.enum || []).length > 0 ? (
                <span style={{ color: 'var(--ant-color-text-tertiary)' }}> &lt;{(p.enum || []).join('|')}&gt;</span>
              ) : null}
              {p.label ? <span style={{ color: 'var(--ant-color-text-quaternary)' }}>  // {p.label}</span> : null}
            </div>
          ))}
          {rels.map(r => (
            <div className='pl-16px' key={r._id}>
              <span style={{ color: 'var(--ant-color-primary-text)' }}>{r.name || '…'}</span>
              {': '}
              <span style={{ color: 'var(--ant-color-info-text)' }}>{r.target_type || '*'}</span>
              {r.cardinality === 'many' ? <span style={{ color: 'var(--ant-color-success-text)' }}>[]</span> : null}
              {(CARD_LABEL[r.cardinality || ''] || r.inverse) ? (
                <span style={{ color: 'var(--ant-color-text-quaternary)' }}>
                  {'  // '}
                  {CARD_LABEL[r.cardinality || ''] || ''}
                  {r.inverse ? ` · inverse ${r.inverse}` : ''}
                </span>
              ) : null}
            </div>
          ))}
        </div>
        <div className='font-mono text-12px leading-22px'>{'}'}</div>
      </div>
    );
  };

  /** A: property rows become a table — column headers carry the semantics
   *  that placeholders used to, and typed tags replace a bare select */
  const propColumns = (tp: TType, typeIdx: number): any[] => [
    {
      dataIndex: 'key',
      title: t('page.ontology.colKey'),
      render: (v: string, p: TProp) => (
        <Input
          onChange={e => {
            const next = (tp.properties || []).map(x => (x._id === p._id ? { ...x, key: e.target.value } : x));
            mutateType(typeIdx, { properties: next });
          }}
          placeholder={t('page.ontology.propKey')}

          status={saveAttempted && vmap[tp._id]?.props[p._id] ? 'error' : undefined}
          value={v}
        />
      )
    },
    {
      dataIndex: 'label',
      title: t('page.ontology.typeLabel'),
      render: (v: string, p: TProp) => (
        <Input
          onChange={e => {
            const next = (tp.properties || []).map(x => (x._id === p._id ? { ...x, label: e.target.value } : x));
            mutateType(typeIdx, { properties: next });
          }}
          placeholder={t('page.ontology.propLabel')}

          value={v}
        />
      )
    },
    {
      dataIndex: 'type',
      title: t('page.ontology.colType'),
      width: 120,
      render: (v: string, p: TProp) => (
        <Select
          onChange={val => {
            const next = (tp.properties || []).map(x => (x._id === p._id ? { ...x, type: val } : x));
            mutateType(typeIdx, { properties: next });
          }}
          options={PROP_TYPES.map(x => ({ value: x, label: <Tag className='!mr-0' color={PROP_TYPE_COLOR[x]}>{x}</Tag> }))}

          value={v}
        />
      )
    },
    {
      dataIndex: 'enum',
      title: t('page.ontology.colEnum'),
      render: (v: string[] | undefined, p: TProp) =>
        p.type === 'enum' ? (
          <Tooltip title={t('page.ontology.enumHint')}>
            <Input
              onChange={e => {
                const next = (tp.properties || []).map(x =>
                  x._id === p._id ? { ...x, enum: e.target.value.split(',').map(s => s.trim()).filter(Boolean) } : x
                );
                mutateType(typeIdx, { properties: next });
              }}
              placeholder='a,b,c'

              value={(p.enum || []).join(',')}
            />
          </Tooltip>
        ) : (
          <span className='text-gray-400'>—</span>
        )
    },
    {
      dataIndex: 'required',
      title: t('page.ontology.required'),
      width: 64,
      render: (v: boolean | undefined, p: TProp) => (
        <Switch
          checked={!!v}
          onChange={val => {
            const next = (tp.properties || []).map(x => (x._id === p._id ? { ...x, required: val } : x));
            mutateType(typeIdx, { properties: next });
          }}

        />
      )
    },
    {
      key: 'del',
      title: '',
      width: 40,
      render: (_: any, p: TProp) => (
        <Button
          danger
          icon={<DeleteOutlined />}
          onClick={() => mutateType(typeIdx, { properties: (tp.properties || []).filter(x => x._id !== p._id) })}

          type='text'
        />
      )
    }
  ];

  const renderTypePanel = (tp: TType, typeIdx: number) => (
    <>
      <div className='mb-12px flex flex-wrap items-center gap-8px'>
        <Input
          className='w-160px'
          onChange={e => renameType(typeIdx, e.target.value)}
          placeholder={t('page.ontology.typeName')}

          status={nameErr(tp)}
          value={tp.name}
        />
        <Input
          className='w-160px'
          onChange={e => mutateType(typeIdx, { label: e.target.value })}
          placeholder={t('page.ontology.typeLabel')}

          value={tp.label}
        />
        <Tooltip title={t('page.ontology.iconHint')}>
          <Input
            className='w-64px'
            onChange={e => mutateType(typeIdx, { icon: e.target.value })}
            placeholder='🙂'

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
      <Input
        className='mb-12px'
        maxLength={200}
        onChange={e => mutateType(typeIdx, { description: e.target.value })}
        placeholder={t('page.ontology.typeDescription')}
        value={tp.description}
      />
      {renderPreview(tp)}
      {/* properties and relations stack vertically: the relation triple card
          needs full width, and side-by-side columns squeeze both sections */}
        {/* properties */}
        <div>
          <div className='mb-2'>
            <span className='font-medium'>{t('page.ontology.properties')}</span>
          </div>
          <Table
            columns={propColumns(tp, typeIdx)}
            dataSource={tp.properties || []}
            locale={{ emptyText: <span className='text-12px text-gray-400'>{t('page.ontology.bareType')}</span> }}
            pagination={false}
            rowKey='_id'
          />
          <Button block className='mt-4px' variant='dashed' icon={<PlusOutlined />} onClick={() => addProperty(typeIdx)}>
            {t('page.ontology.addProperty')}
          </Button>
        </div>

        {/* relations — direction triples: source chip — name → target, so the
            semantics hidden in selects (direction, cardinality, inverse) stay
            visible on the surface */}
        <div className='mt-20px'>
          <div className='mb-2'>
            <span className='font-medium'>{t('page.ontology.relations')}</span>
          </div>
          {(tp.relations || []).map((rel: TRel) => (
            <div
              className='mb-6px flex flex-wrap items-center gap-6px rounded-6px border border-solid px-10px py-8px'
              key={rel._id}
              style={{ borderColor: 'var(--ant-color-border-secondary)' }}
            >
              <Tag className='!mr-0' color='purple'>
                {tp.name || '…'}
              </Tag>
              <span className='text-gray-400'>—</span>
              <Input
                className='w-160px'
                onChange={e => {
                  const next = (tp.relations || []).map(r => (r._id === rel._id ? { ...r, name: e.target.value } : r));
                  mutateType(typeIdx, { relations: next });
                }}
                placeholder={t('page.ontology.relName')}

                status={saveAttempted && vmap[tp._id]?.rels[rel._id] ? 'error' : undefined}
                value={rel.name}
              />
              <ArrowRightOutlined className='text-gray-400' />
              <Select
                className='w-220px'
                onChange={v => {
                  const next = (tp.relations || []).map(r => (r._id === rel._id ? { ...r, target_type: v } : r));
                  mutateType(typeIdx, { relations: next });
                }}
                options={[{ value: '*', label: '* — 任意类型' }, ...types.filter(x => x.name).map(x => ({ value: x.name, label: x.name }))]}
                showSearch

                value={rel.target_type}
              />
              <Select
                allowClear
                className='w-130px'
                onChange={v => {
                  const next = (tp.relations || []).map(r => (r._id === rel._id ? { ...r, cardinality: v } : r));
                  mutateType(typeIdx, { relations: next });
                }}
                options={[{ value: 'one', label: CARD_LABEL.one }, { value: 'many', label: CARD_LABEL.many }]}
                placeholder={t('page.ontology.cardinality')}

                value={rel.cardinality}
              />
              <span className='text-12px text-gray-500'>{t('page.ontology.inverse')}</span>
              <Input
                className='w-160px'
                onChange={e => {
                  const next = (tp.relations || []).map(r => (r._id === rel._id ? { ...r, inverse: e.target.value } : r));
                  mutateType(typeIdx, { relations: next });
                }}
                placeholder={t('page.ontology.inverse')}

                value={rel.inverse}
              />
              <Button
                className='ml-auto'
                danger
                icon={<DeleteOutlined />}
                onClick={() => mutateType(typeIdx, { relations: (tp.relations || []).filter(r => r._id !== rel._id) })}

                type='text'
              />
            </div>
          ))}
          {!(tp.relations || []).length && (
            <div className='text-12px text-gray-400'>{t('page.ontology.bareType')}</div>
          )}
          <Button block className='mt-8px' variant='dashed' icon={<PlusOutlined />} onClick={() => addRelation(typeIdx)}>
            {t('page.ontology.addRelation')}
          </Button>
        </div>
      {/* page layout: the section list a generated page follows */}
      <div className='mt-20px'>
        <div className='mb-2'>
          <span className='font-medium'>{t('page.ontology.pageSections')}</span>
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

              value={sec.key}
            />
            <Input
              className='w-180px'
              onChange={e => {
                const next = (tp.page_sections || []).map(s => (s._id === sec._id ? { ...s, title: e.target.value } : s));
                mutateType(typeIdx, { page_sections: next });
              }}
              placeholder={t('page.ontology.sectionTitle')}

              value={sec.title}
            />
            <Button
              danger
              icon={<DeleteOutlined />}
              onClick={() => mutateType(typeIdx, { page_sections: (tp.page_sections || []).filter(s => s._id !== sec._id) })}

              type='text'
            />
          </div>
        ))}
        <Button block className='mt-4px' variant='dashed' icon={<PlusOutlined />} onClick={() => addPageSection(typeIdx)}>
          {t('page.ontology.addSection')}
        </Button>
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
                  <div className='flex min-w-0 items-center gap-8px'>
                    <span>{tp.icon}</span>
                    <span className={saveAttempted && vmap[tp._id]?.name ? 'color-[var(--ant-color-error)] font-medium' : 'font-medium'}>
                      {tp.name || t('page.ontology.typeName')}
                    </span>
                    {tp.label && <span className='text-gray-500'>{tp.label}</span>}
                    {tp.description && (
                      <Tooltip title={tp.description}>
                        <span className='hidden min-w-0 max-w-320px truncate text-12px text-gray-400 xl:!inline'>
                          {tp.description}
                        </span>
                      </Tooltip>
                    )}
                    {/* at-a-glance schema chips: readable even when collapsed */}
                    <span className='hidden min-w-0 flex-1 items-center gap-4px overflow-hidden lg:!flex'>
                      {(tp.properties || []).slice(0, 3).map(p => (
                        <span
                          className='whitespace-nowrap rounded-4px px-6px py-1px font-mono text-11px'
                          key={p._id}
                          style={{ background: 'var(--ant-color-fill-tertiary)', color: 'var(--ant-color-text-secondary)' }}
                        >
                          {p.key}
                          {p.required ? '*' : ''}:{p.type}
                        </span>
                      ))}
                      {(tp.relations || []).slice(0, 2).map(r => (
                        <span
                          className='whitespace-nowrap rounded-4px px-6px py-1px text-11px'
                          key={r._id}
                          style={{ background: 'var(--ant-color-fill-tertiary)', color: 'var(--ant-color-text-secondary)' }}
                        >
                          {r.name} → {r.target_type}
                        </span>
                      ))}
                      {(tp.properties?.length || 0) + (tp.relations?.length || 0) > 5 && (
                        <span className='whitespace-nowrap text-11px' style={{ color: 'var(--ant-color-text-quaternary)' }}>
                          +{(tp.properties?.length || 0) + (tp.relations?.length || 0) - 5}
                        </span>
                      )}
                    </span>
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
