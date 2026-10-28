import { Button, Divider, Form, Input, InputNumber, Select, Spin, Switch, Tabs } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import '../index.scss';
import { fetchIntegration, updateIntegration } from '@/service/api/integration';
import { fetchSettings, updateSettings } from '@/service/api/server';
import { backfillWikiProjection } from '@/service/api';
import DataSourceSelect from '@/pages/data-source/modules/DataSourceSelect';
import { DocTableSelect } from './DocPicker';
import BlockedPatternsField from './BlockedPatternsField';
import RecommendTopicsField from './RecommendTopicsField';
import RsaSourcesSection from './RsaSourcesSection';
import AggregationFieldsField, { AGG_DIMENSIONS } from './AggregationFieldsField';
import { useLoading, useRequest } from '@sa/hooks';
import { getApplicationSetting, setApplicationSetting, updateRootRouteIfSearch } from '@/store/slice/server';
import { initAuthRoute, initConstantRoute, selectFilterPaths, setFilterPaths } from '@/store/slice/route';
import { resetAuth } from '@/store/slice/auth';
import BuiltinSearchForm from './BuiltinSearchForm';
import SaveBar from './SaveBar';

// the built-in AI search integration (backend: core.DefaultSearchIntegrationID).
// the app's own search page always reuses this fullscreen widget — it is hidden
// from the integration list, and this settings tab is its only editing UI
const BUILTIN_SEARCH_INTEGRATION_ID = 'full-screen-widget-default';

// mirrors core.DefaultFieldWeights — shown as the editable starting rows when
// the operator hasn't saved an explicit field_weights list yet
const DEFAULT_FIELD_WEIGHTS = [
  { field: 'title.keyword', weight: 100 },
  { field: 'title', weight: 10 },
  { field: 'title.pinyin', weight: 4 },
  { field: 'combined_fulltext', weight: 1 }
];

// normalizes one saved weight row for the Form.List editor
const normalizeWeightRow = (row: any) => ({
  field: row?.field || '',
  weight: Number(row?.weight) || 1
});

// known _source fields offered as dropdown candidates for the result-field
// whitelist (result_source_includes). mode="tags" still accepts arbitrary
// custom names, so this list is a convenience, not a constraint.
const RESULT_FIELD_CANDIDATES: Record<string, string[]> = {
  identity: ['id', 'title', 'summary', 'url', 'icon', 'thumbnail', 'cover'],
  classify: ['type', 'category', 'subcategory', 'categories', 'tags'],
  body: ['content', 'lang'],
  ownership: ['owner', 'source', 'connector_name']
};

// the five panes — also the `?tab=` seeds accepted on this route
type SearchSettingsTab = 'basics' | 'retrieval' | 'sources' | 'results' | 'security';
const TAB_KEYS: SearchSettingsTab[] = ['basics', 'retrieval', 'sources', 'results', 'security'];

// field → owning pane, used to jump to the failing tab when the save's
// validateFields rejects (the bad field may live on a hidden pane)
const FIELD_TABS: Record<string, SearchSettingsTab> = {
  enabled: 'basics',
  search_type: 'basics',
  wiki_projection: 'retrieval',
  chunk_route: 'retrieval',
  default_sort: 'retrieval',
  match_operator: 'retrieval',
  minimum_should_match: 'retrieval',
  min_score: 'retrieval',
  field_weights: 'retrieval',
  semantic_min_score: 'retrieval',
  rrf_rank_constant: 'retrieval',
  rerank_enabled: 'retrieval',
  max_result_window: 'retrieval',
  fuzziness: 'retrieval',
  query_timeout_ms: 'retrieval',
  rsa_enabled: 'sources',
  rsa_max_sources_per_query: 'sources',
  rsa_cache_size: 'sources',
  page_size: 'results',
  max_page_size: 'results',
  result_source_includes: 'results',
  suggest_enabled: 'results',
  suggest_size: 'results',
  hot_searches_enabled: 'results',
  hot_searches_size: 'results',
  recommend_topics: 'results',
  recommend_topics_enabled: 'results',
  related_searches_enabled: 'results',
  aggregations_enabled: 'results',
  aggregation_fields_disabled: 'results',
  aggregation_bucket_sort: 'results',
  aggregation_bucket_cap: 'results',
  blocked_doc_ids: 'security',
  blocked_datasource_ids: 'security',
  pinned_rules: 'security',
  apply_masking_to_results: 'security',
  blocked_query_patterns: 'security'
};

const SearchSettings = memo(() => {
  const [form] = Form.useForm();
  const recommendEnabled = Form.useWatch(['recommend_topics_enabled'], form);
  const { t } = useTranslation();

  // page-level SaveBar state: `dirty` covers the search_settings form here,
  // `widgetDirty` is raised by BuiltinSearchForm; one bar saves both sections
  const [dirty, setDirty] = useState(false);
  const [widgetDirty, setWidgetDirty] = useState(false);
  // bumping the key remounts BuiltinSearchForm so its record-load effect
  // re-applies everything (its fields live in its own form + local states)
  const [widgetResetKey, setWidgetResetKey] = useState(0);
  const widgetSubmitRef = useRef<null | (() => Promise<void>)>(null);

  // one-off wiki projection backfill — the preparation step the
  // wiki_projection toggle's hint refers to (article-update permission,
  // mirroring the endpoint's gate)
  const [backfilling, setBackfilling] = useState(false);
  const onBackfillProjection = () => {
    setBackfilling(true);
    backfillWikiProjection()
      .then(res => {
        window.$message?.success(
          t('page.settings.search_settings.labels.backfill_done', { projected: res?.projected ?? 0, total: res?.total ?? 0 })
        );
      })
      .finally(() => setBackfilling(false));
  };

  // `?tab=` seeds the requested tab. Tab clicks stay local-state on purpose:
  // routing them through setSearchParams trips the hash router's navigation
  // blocker and bounces to login (same rationale as the settings page)
  const [searchParams] = useSearchParams();
  const requestedTab = searchParams.get('tab');
  const [activeTab, setActiveTab] = useState<string>(() =>
    requestedTab && (TAB_KEYS as string[]).includes(requestedTab) ? requestedTab : 'basics'
  );

  // the component stays mounted across hash-query navigations (same route),
  // so the useState initializer alone can't pick up later ?tab= changes
  useEffect(() => {
    if (requestedTab && (TAB_KEYS as string[]).includes(requestedTab)) {
      setActiveTab(requestedTab);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [requestedTab]);

  const { endLoading, loading, startLoading } = useLoading();

  const dispatch = useAppDispatch();
  const applicationSetting = useAppSelector(getApplicationSetting);
  const filterPaths = useAppSelector(selectFilterPaths);

  const {
    data,
    loading: dataLoading,
    run
  } = useRequest(fetchSettings, {
    manual: true
  });

  // the built-in fullscreen widget whose parameters this page edits through
  // the same form components the integration editor uses
  const {
    data: builtin,
    loading: builtinLoading,
    run: runBuiltin
  } = useRequest(fetchIntegration, {
    manual: true
  });

  useEffect(() => {
    run();
  }, []);

  useMount(() => {
    run();
    runBuiltin(BUILTIN_SEARCH_INTEGRATION_ID);
  });

  const saveSearchSettings = async ({ silent = false }: { silent?: boolean } = {}) => {
    let params;
    try {
      params = await form.validateFields();
    } catch (err: any) {
      // surface the failing pane: jump to the tab owning the first bad field
      const firstKey = err?.errorFields?.[0]?.name?.[0];
      const tab = firstKey ? FIELD_TABS[firstKey] : undefined;
      if (tab) setActiveTab(tab);
      throw err;
    }
    const { enabled, search_type } = params;
    startLoading();
    const search_settings = {
      enabled,
      wiki_projection: form.getFieldValue('wiki_projection') ?? false,
      chunk_route: form.getFieldValue('chunk_route') ?? false,
      // federated leg (RSA) — 0 keeps the backend defaults (8 sources / 256 entries)
      rsa_enabled: form.getFieldValue('rsa_enabled') ?? false,
      rsa_max_sources_per_query: Number(params.rsa_max_sources_per_query) || 0,
      rsa_cache_size: Number(params.rsa_cache_size) || 0,
      search_type: search_type || 'hybrid_rrf',
      // retrieval strategy — empty strings keep the backend fallbacks in place
      default_sort: params.default_sort || 'relevance',
      match_operator: params.match_operator || 'OR',
      minimum_should_match: (params.minimum_should_match || '').trim(),
      min_score: Number(params.min_score) || 0,
      field_weights: (params.field_weights || [])
        .filter((row: any) => row?.field?.trim())
        .map((row: any) => ({ field: row.field.trim(), weight: Number(row.weight) || 1 })),
      // presentation / suggestions / aggregation / protections — 0 keeps the
      // backend defaults; suggest_disabled is the inverted storage of the
      // suggest_enabled switch (zero value = enabled, the historic behavior)
      page_size: Number(params.page_size) || 0,
      max_page_size: Number(params.max_page_size) || 0,
      max_result_window: Number(params.max_result_window) || 0,
      suggest_disabled: !params.suggest_enabled,
      suggest_size: Number(params.suggest_size) || 0,
      aggregation_bucket_cap: Number(params.aggregation_bucket_cap) || 0,
      result_source_includes: (params.result_source_includes || []).map((v: string) => v.trim()).filter(Boolean),
      fuzziness: Number(params.fuzziness) || 0,
      query_timeout_ms: Number(params.query_timeout_ms) || 0,
      aggregations_disabled: !params.aggregations_enabled,
      aggregation_fields_disabled: (params.aggregation_fields_disabled || []).filter((v: string) =>
        (AGG_DIMENSIONS as readonly string[]).includes(v)
      ),
      aggregation_bucket_sort: params.aggregation_bucket_sort || 'count',
      hot_searches_disabled: !params.hot_searches_enabled,
      hot_searches_size: Number(params.hot_searches_size) || 0,
      recommend_topics: (params.recommend_topics || []).map((v: string) => v.trim()).filter(Boolean),
      recommend_topics_disabled: !params.recommend_topics_enabled,
      related_searches_disabled: !params.related_searches_enabled,
      // semantic / AI — rerank_disabled is the inverted rerank_enabled switch
      semantic_min_score: Number(params.semantic_min_score) || 0,
      rrf_rank_constant: Number(params.rrf_rank_constant) || 0,
      rerank_disabled: !params.rerank_enabled,
      // result intervention
      blocked_doc_ids: (params.blocked_doc_ids || []).map((v: string) => v.trim()).filter(Boolean),
      blocked_datasource_ids: (params.blocked_datasource_ids || []).map((v: string) => v.trim()).filter(Boolean),
      pinned_rules: (params.pinned_rules || [])
        .filter((rule: any) => rule?.query?.trim() && rule?.document_ids?.length)
        .map((rule: any) => ({ query: rule.query.trim(), document_ids: rule.document_ids })),
      // security
      apply_masking_to_results: !!params.apply_masking_to_results,
      blocked_query_patterns: (params.blocked_query_patterns || []).map((v: string) => v.trim()).filter(Boolean)
    }
    const result = await updateSettings({
       search_settings
    });
    if (result?.data?.acknowledged) {
      const newApplicationSetting = {
        ...applicationSetting,
        search_settings
      }
      await dispatch(setApplicationSetting(newApplicationSetting));
      await dispatch(updateRootRouteIfSearch(newApplicationSetting));
      if (search_settings.enabled) {
        await dispatch(setFilterPaths(filterPaths.filter(path => path !== '/search')));
      }
      await dispatch(initConstantRoute());
      await dispatch(resetAuth());
      await dispatch(initAuthRoute());
      if (!silent) {
        window.$message?.success(t('common.updateSuccess'));
      }
    }
    endLoading();
  };

  // same submit contract as the integration edit page — the payload lands on
  // the built-in integration, so the app search and an embedded widget behave
  // identically
  const handleUpdateBuiltin = async (params: any, before?: () => void, after?: () => void) => {
    if (before) before();
    const res = await updateIntegration({ id: BUILTIN_SEARCH_INTEGRATION_ID, ...params });
    if (res?.data?.result === 'updated') {
      window.$message?.success(t('common.updateSuccess'));
    }
    if (after) after();
  };

  const applySearchData = (payload: any) => {
    if (payload?.search_settings) {
      form.setFieldsValue({
        ...payload.search_settings,
        // retired 'hybrid' migrates to hybrid_rrf on display, so the select
        // always shows a real label and the next save persists the new value
        search_type: payload.search_settings.search_type === 'keyword' || payload.search_settings.search_type === 'semantic'
          ? payload.search_settings.search_type
          : 'hybrid_rrf',
        default_sort: payload.search_settings.default_sort || 'relevance',
        match_operator: payload.search_settings.match_operator || 'OR',
        minimum_should_match: payload.search_settings.minimum_should_match || '',
        min_score: payload.search_settings.min_score ?? 0,
        field_weights: payload.search_settings.field_weights?.length
          ? payload.search_settings.field_weights.map(normalizeWeightRow)
          : DEFAULT_FIELD_WEIGHTS.map(normalizeWeightRow),
        // presentation / suggestions / aggregation / protections — effective
        // defaults so the form always shows meaningful numbers
        page_size: payload.search_settings.page_size || 10,
        max_page_size: payload.search_settings.max_page_size || 100,
        max_result_window: payload.search_settings.max_result_window || 10000,
        suggest_enabled: !payload.search_settings.suggest_disabled,
        suggest_size: payload.search_settings.suggest_size || 10,
        aggregation_bucket_cap: payload.search_settings.aggregation_bucket_cap || 15,
        result_source_includes: payload.search_settings.result_source_includes || [],
        fuzziness: payload.search_settings.fuzziness || 3,
        query_timeout_ms: payload.search_settings.query_timeout_ms ?? 0,
        aggregations_enabled: !payload.search_settings.aggregations_disabled,
        aggregation_fields_disabled: payload.search_settings.aggregation_fields_disabled || [],
        aggregation_bucket_sort: payload.search_settings.aggregation_bucket_sort || 'count',
        hot_searches_enabled: !payload.search_settings.hot_searches_disabled,
        hot_searches_size: payload.search_settings.hot_searches_size || 10,
        recommend_topics: payload.search_settings.recommend_topics || [],
        recommend_topics_enabled: !payload.search_settings.recommend_topics_disabled,
        related_searches_enabled: !payload.search_settings.related_searches_disabled,
        // semantic / AI + intervention + security
        semantic_min_score: payload.search_settings.semantic_min_score ?? 0,
        rrf_rank_constant: payload.search_settings.rrf_rank_constant ?? 0,
        // RSA federated leg — defaults mirror the backend fallbacks so the
        // pane shows the effective budget even before the first explicit save
        rsa_enabled: !!payload.search_settings.rsa_enabled,
        rsa_max_sources_per_query: payload.search_settings.rsa_max_sources_per_query || 8,
        rsa_cache_size: payload.search_settings.rsa_cache_size || 256,
        rerank_enabled: !payload.search_settings.rerank_disabled,
        blocked_doc_ids: payload.search_settings.blocked_doc_ids || [],
        blocked_datasource_ids: payload.search_settings.blocked_datasource_ids || [],
        pinned_rules: payload.search_settings.pinned_rules?.length
          ? payload.search_settings.pinned_rules
          : [],
        apply_masking_to_results: !!payload.search_settings.apply_masking_to_results,
        blocked_query_patterns: payload.search_settings.blocked_query_patterns || []
      });
    } else {
      form.setFieldsValue({
        enabled: false,
        search_type: 'hybrid_rrf',
        default_sort: 'relevance',
        match_operator: 'OR',
        minimum_should_match: '',
        min_score: 0,
        field_weights: DEFAULT_FIELD_WEIGHTS.map(normalizeWeightRow),
        page_size: 10,
        max_page_size: 100,
        max_result_window: 10000,
        suggest_enabled: true,
        suggest_size: 10,
        aggregation_bucket_cap: 15,
        result_source_includes: [],
        fuzziness: 3,
        query_timeout_ms: 0,
        aggregations_enabled: true,
        aggregation_fields_disabled: [],
        aggregation_bucket_sort: 'count',
        hot_searches_enabled: true,
        hot_searches_size: 10,
        recommend_topics: [],
        recommend_topics_enabled: true,
        related_searches_enabled: true,
        semantic_min_score: 0,
        rrf_rank_constant: 0,
        rsa_enabled: false,
        rsa_max_sources_per_query: 8,
        rsa_cache_size: 256,
        rerank_enabled: true,
        blocked_doc_ids: [],
        blocked_datasource_ids: [],
        pinned_rules: [],
        apply_masking_to_results: false,
        blocked_query_patterns: []
      });
    }
  };

  useEffect(() => {
    applySearchData(data);
    setDirty(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(data)]);

  // one save for both sections: search_settings first (its validateFields
  // rejection aborts before the integration update), then the built-in widget
  const handleSaveAll = async () => {
    try {
      await saveSearchSettings();
    } catch {
      // validation errors are surfaced on the fields — keep the bar dirty
      return;
    }
    try {
      await widgetSubmitRef.current?.();
      setDirty(false);
      setWidgetDirty(false);
    } catch {
      // the builtin widget form failed validation — show its pane
      setActiveTab('basics');
    }
  };

  const handleDiscard = () => {
    setDirty(false);
    setWidgetDirty(false);
    applySearchData(data);
    setWidgetResetKey(key => key + 1);
  };

  return (
    <ListContainer>
      <Spin spinning={dataLoading || loading}>
        <Form
          className="settings-form pb-24px"
          colon={false}
          form={form}
          labelAlign="left"
          onValuesChange={() => setDirty(true)}
        >
          {/* 4-tab layout. forceRender keeps every pane mounted so both forms
              (search_settings + the builtin widget) retain unsaved state across
              switches and the single SaveBar below still owns everything */}
          <Tabs
            activeKey={activeTab}
            className="search-settings-tabs"
            items={[
              {
                key: 'basics',
                label: t('page.settings.search_settings.tabs.basics'),
                forceRender: true,
                children: (
                  <div className="search-tab-pane">
                    <div className="color-[var(--ant-color-text)] font-medium mb-8px">
                      {t('page.settings.search_settings.labels.builtin_widget')}
                    </div>
                    <div className="mb-24px settings-form-help">
                      {t('page.settings.search_settings.labels.builtin_widget_hint')}
                    </div>
                    <Form.Item
                        label={t('page.settings.search_settings.labels.enabled')}
                        name={['enabled']}
                        valuePropName="checked"
                      >
                      <Switch size="small" />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.search_type')}
                      name={['search_type']}
                      tooltip={t('page.settings.search_settings.labels.search_type_desc')}
                    >
                      <Select
                        className="!w-520px"
                        // plain weighted 'hybrid' is retired — one hybrid entry
                        // (RRF fusion) covers it; hybrid_rrf first as the default
                        options={['hybrid_rrf', 'keyword', 'semantic'].map((value) => ({
                          value,
                          label: t(`page.settings.search_settings.options.${value}`)
                        }))}
                      />
                    </Form.Item>

                    {
                      builtin ? (
                        <BuiltinSearchForm
                          key={widgetResetKey}
                          loading={builtinLoading}
                          record={builtin?._source}
                          onDirtyChange={setWidgetDirty}
                          onRegisterSubmit={fn => {
                            widgetSubmitRef.current = fn;
                          }}
                          onSubmit={handleUpdateBuiltin}
                        />
                      ) : (
                        <Spin />
                      )
                    }
                  </div>
                )
              },
              {
                key: 'retrieval',
                label: t('page.settings.search_settings.tabs.retrieval'),
                forceRender: true,
                children: (
                  <div className="search-tab-pane">
                    <div className="color-[var(--ant-color-text)] font-medium mb-8px">
                      {t('page.settings.search_settings.labels.strategy')}
                    </div>
                    <div className="mb-16px settings-form-help">
                      {t('page.settings.search_settings.labels.strategy_hint')}
                    </div>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.default_sort')}
                      name={['default_sort']}
                      tooltip={t('page.settings.search_settings.labels.default_sort_desc')}
                    >
                      <Select
                        className="!w-520px"
                        options={['relevance', 'created_desc', 'created_asc'].map((value) => ({
                          value,
                          label: t(`page.settings.search_settings.options.${value}`)
                        }))}
                      />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.match_operator')}
                      name={['match_operator']}
                      tooltip={t('page.settings.search_settings.labels.match_operator_desc')}
                    >
                      <Select
                        className="!w-520px"
                        options={[
                          { value: 'OR', label: t('page.settings.search_settings.options.match_or') },
                          { value: 'AND', label: t('page.settings.search_settings.options.match_and') }
                        ]}
                      />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.minimum_should_match')}
                      name={['minimum_should_match']}
                      tooltip={t('page.settings.search_settings.labels.minimum_should_match_desc')}
                    >
                      <Input className="w-520px" placeholder="75%" />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.min_score')}
                      name={['min_score']}
                      tooltip={t('page.settings.search_settings.labels.min_score_desc')}
                    >
                      <InputNumber className="w-520px" max={10000} min={0} />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.field_weights')}
                      tooltip={t('page.settings.search_settings.labels.field_weights_desc')}
                    >
                      <Form.List name={['field_weights']}>
                        {(fields, { add, remove }) => (
                          <div className="flex flex-col gap-8px">
                            {fields.map((field) => (
                              <div key={field.key} className="flex items-start gap-8px">
                                <Form.Item
                                  className="mb-0 flex-1"
                                  name={[field.name, 'field']}
                                  rules={[{ required: true, message: t('page.settings.search_settings.labels.field') }]}
                                >
                                  <Input placeholder="title" />
                                </Form.Item>
                                <Form.Item
                                  className="mb-0"
                                  name={[field.name, 'weight']}
                                  rules={[{ required: true, message: t('page.settings.search_settings.labels.weight') }]}
                                >
                                  <InputNumber className="w-120px" max={10000} min={0.0001} />
                                </Form.Item>
                                <Button
                                  disabled={fields.length <= 1}
                                  icon={<DeleteOutlined />}
                                  onClick={() => remove(field.name)}
                                  size="small"
                                  type="text"
                                />
                              </div>
                            ))}
                            <Button
                              className="self-start"
                              icon={<PlusOutlined />}
                              onClick={() => add({ field: '', weight: 1 })}
                              size="small"
                            >
                              {t('page.settings.search_settings.labels.add_field_weight')}
                            </Button>
                          </div>
                        )}
                      </Form.List>
                    </Form.Item>

                    <Divider />

                    <div className="color-[var(--ant-color-text)] font-medium mb-8px">
                      {t('page.settings.search_settings.labels.semantic_ai')}
                    </div>
                    <div className="mb-16px settings-form-help">
                      {t('page.settings.search_settings.labels.semantic_ai_hint')}
                    </div>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.wiki_projection')}
                      name={['wiki_projection']}
                      valuePropName="checked"
                      tooltip={t('page.settings.search_settings.labels.wiki_projection_desc')}
                    >
                      <Switch size="small" />
                    </Form.Item>
                    <div className='mb-16px'>
                      <Button
                        loading={backfilling}
                        onClick={onBackfillProjection}
                        size="small"
                      >
                        {t('page.settings.search_settings.labels.backfill_action')}
                      </Button>
                    </div>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.chunk_route')}
                      name={['chunk_route']}
                      valuePropName="checked"
                      tooltip={t('page.settings.search_settings.labels.chunk_route_desc')}
                    >
                      <Switch size="small" />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.semantic_min_score')}
                      name={['semantic_min_score']}
                      tooltip={t('page.settings.search_settings.labels.semantic_min_score_desc')}
                    >
                      <InputNumber className="w-520px" max={10000} min={0} step={0.01} />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.rrf_rank_constant')}
                      name={['rrf_rank_constant']}
                      tooltip={t('page.settings.search_settings.labels.rrf_rank_constant_desc')}
                    >
                      <InputNumber className="w-520px" max={1000} min={0} />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.rerank_enabled')}
                      name={['rerank_enabled']}
                      valuePropName="checked"
                      tooltip={t('page.settings.search_settings.labels.rerank_enabled_desc')}
                    >
                      <Switch size="small" />
                    </Form.Item>

                    <Divider />

                    <div className="color-[var(--ant-color-text)] font-medium mb-8px">
                      {t('page.settings.search_settings.labels.advanced')}
                    </div>
                    <div className="mb-16px settings-form-help">
                      {t('page.settings.search_settings.labels.advanced_hint')}
                    </div>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.max_result_window')}
                      name={['max_result_window']}
                      tooltip={t('page.settings.search_settings.labels.max_result_window_desc')}
                    >
                      <InputNumber className="w-520px" max={100000} min={100} />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.fuzziness')}
                      name={['fuzziness']}
                      tooltip={t('page.settings.search_settings.labels.fuzziness_desc')}
                    >
                      <InputNumber className="w-520px" max={5} min={1} />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.query_timeout_ms')}
                      name={['query_timeout_ms']}
                      tooltip={t('page.settings.search_settings.labels.query_timeout_ms_desc')}
                    >
                      <InputNumber className="w-520px" max={60000} min={0} step={100} />
                    </Form.Item>
                  </div>
                )
              },
              {
                key: 'sources',
                label: t('page.settings.search_settings.tabs.sources'),
                forceRender: true,
                children: (
                  <div className="search-tab-pane">
                    <div className="color-[var(--ant-color-text)] font-medium mb-8px">
                      {t('page.settings.search_settings.labels.rsa_global')}
                    </div>
                    <div className="mb-16px settings-form-help">
                      {t('page.settings.search_settings.labels.rsa_global_hint')}
                    </div>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.rsa_enabled')}
                      name={['rsa_enabled']}
                      valuePropName="checked"
                      tooltip={t('page.settings.search_settings.labels.rsa_enabled_desc')}
                    >
                      <Switch size="small" />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.rsa_max_sources')}
                      name={['rsa_max_sources_per_query']}
                      tooltip={t('page.settings.search_settings.labels.rsa_max_sources_desc')}
                    >
                      <InputNumber className="w-520px" max={64} min={1} />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.rsa_cache_size')}
                      name={['rsa_cache_size']}
                      tooltip={t('page.settings.search_settings.labels.rsa_cache_size_desc')}
                    >
                      <InputNumber className="w-520px" max={4096} min={0} />
                    </Form.Item>

                    <Divider />

                    <RsaSourcesSection />
                  </div>
                )
              },
              {
                key: 'results',
                label: t('page.settings.search_settings.tabs.results'),
                forceRender: true,
                children: (
                  <div className="search-tab-pane">
                    <div className="color-[var(--ant-color-text)] font-medium mb-8px">
                      {t('page.settings.search_settings.labels.presentation')}
                    </div>
                    <div className="mb-16px settings-form-help">
                      {t('page.settings.search_settings.labels.presentation_hint')}
                    </div>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.page_size')}
                      name={['page_size']}
                      tooltip={t('page.settings.search_settings.labels.page_size_desc')}
                    >
                      <InputNumber className="w-520px" max={100} min={1} />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.max_page_size')}
                      name={['max_page_size']}
                      tooltip={t('page.settings.search_settings.labels.max_page_size_desc')}
                    >
                      <InputNumber className="w-520px" max={1000} min={10} />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.result_source_includes')}
                      name={['result_source_includes']}
                      tooltip={t('page.settings.search_settings.labels.result_source_includes_desc')}
                    >
                      <Select
                        className="max-w-560px"
                        mode="tags"
                        placeholder="title, summary"
                        options={Object.entries(RESULT_FIELD_CANDIDATES).map(([key, fields]) => ({
                          label: t(`page.settings.search_settings.labels.field_group_${key}`),
                          options: fields.map(f => ({ value: f, label: f }))
                        }))}
                      />
                    </Form.Item>

                    <Divider />

                    <div className="color-[var(--ant-color-text)] font-medium mb-8px">
                      {t('page.settings.search_settings.labels.suggestions')}
                    </div>
                    <div className="mb-16px settings-form-help">
                      {t('page.settings.search_settings.labels.suggestions_hint')}
                    </div>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.suggest_enabled')}
                      name={['suggest_enabled']}
                      valuePropName="checked"
                      tooltip={t('page.settings.search_settings.labels.suggest_enabled_desc')}
                    >
                      <Switch size="small" />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.suggest_size')}
                      name={['suggest_size']}
                      tooltip={t('page.settings.search_settings.labels.suggest_size_desc')}
                    >
                      <InputNumber className="w-520px" max={50} min={1} />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.hot_searches_enabled')}
                      name={['hot_searches_enabled']}
                      valuePropName="checked"
                      tooltip={t('page.settings.search_settings.labels.hot_searches_enabled_desc')}
                    >
                      <Switch size="small" />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.hot_searches_size')}
                      name={['hot_searches_size']}
                      tooltip={t('page.settings.search_settings.labels.hot_searches_size_desc')}
                    >
                      <InputNumber className="w-520px" max={20} min={1} />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.recommend_topics_enabled')}
                      name={['recommend_topics_enabled']}
                      valuePropName="checked"
                      tooltip={t('page.settings.search_settings.labels.recommend_topics_enabled_desc')}
                    >
                      <Switch size="small" />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.recommend_topics')}
                      name={['recommend_topics']}
                      tooltip={t('page.settings.search_settings.labels.recommend_topics_desc')}
                    >
                      <RecommendTopicsField disabled={recommendEnabled === false} />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.related_searches_enabled')}
                      name={['related_searches_enabled']}
                      valuePropName="checked"
                      tooltip={t('page.settings.search_settings.labels.related_searches_enabled_desc')}
                    >
                      <Switch size="small" />
                    </Form.Item>

                    <Divider />

                    <div className="color-[var(--ant-color-text)] font-medium mb-8px">
                      {t('page.settings.search_settings.labels.aggregations')}
                    </div>
                    <div className="mb-16px settings-form-help">
                      {t('page.settings.search_settings.labels.aggregations_hint')}
                    </div>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.aggregations_enabled')}
                      name={['aggregations_enabled']}
                      valuePropName="checked"
                      tooltip={t('page.settings.search_settings.labels.aggregations_enabled_desc')}
                    >
                      <Switch size="small" />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.aggregation_bucket_sort')}
                      name={['aggregation_bucket_sort']}
                      tooltip={t('page.settings.search_settings.labels.aggregation_bucket_sort_desc')}
                    >
                      <Select
                        className="!w-520px"
                        options={[
                          { value: 'count', label: t('page.settings.search_settings.options.aggregation_bucket_sort_count') },
                          { value: 'value', label: t('page.settings.search_settings.options.aggregation_bucket_sort_value') }
                        ]}
                      />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.aggregation_bucket_cap')}
                      name={['aggregation_bucket_cap']}
                      tooltip={t('page.settings.search_settings.labels.aggregation_bucket_cap_desc')}
                    >
                      <InputNumber className="w-520px" max={50} min={1} />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.aggregation_fields')}
                      name={['aggregation_fields_disabled']}
                      tooltip={t('page.settings.search_settings.labels.aggregation_fields_desc')}
                    >
                      <AggregationFieldsField />
                    </Form.Item>
                  </div>
                )
              },
              {
                key: 'security',
                label: t('page.settings.search_settings.tabs.security'),
                forceRender: true,
                children: (
                  <div className="search-tab-pane">
                    <div className="color-[var(--ant-color-text)] font-medium mb-8px">
                      {t('page.settings.search_settings.labels.intervention')}
                    </div>
                    <div className="mb-16px settings-form-help">
                      {t('page.settings.search_settings.labels.intervention_hint')}
                    </div>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.blocked_doc_ids')}
                      name={['blocked_doc_ids']}
                      tooltip={t('page.settings.search_settings.labels.blocked_doc_ids_desc')}
                    >
                      <DocTableSelect />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.blocked_datasource_ids')}
                      name={['blocked_datasource_ids']}
                      tooltip={t('page.settings.search_settings.labels.blocked_datasource_ids_desc')}
                      getValueProps={(ids?: string[]) => ({ value: (ids || []).map((id) => ({ id })) })}
                      getValueFromEvent={(rows: any[]) => (rows || []).map((r: any) => (typeof r === 'string' ? r : r.id))}
                    >
                      <DataSourceSelect mode="multiple" className="max-w-560px" />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.pinned_rules')}
                      tooltip={t('page.settings.search_settings.labels.pinned_rules_desc')}
                    >
                      <Form.List name={['pinned_rules']}>
                        {(fields, { add, remove }) => (
                          <div className="flex flex-col gap-12px w-full">
                            {fields.map((field) => (
                              <div
                                key={field.key}
                                className="rounded-8px p-12px flex flex-col gap-8px"
                                style={{ border: '1px solid var(--ant-color-border-secondary)' }}
                              >
                                <div className="flex items-start gap-8px">
                                  <Form.Item
                                    className="mb-0 flex-1"
                                    name={[field.name, 'query']}
                                    rules={[{ required: true, message: t('page.settings.search_settings.labels.pinned_query') }]}
                                  >
                                    <Input placeholder={t('page.settings.search_settings.labels.pinned_query')} />
                                  </Form.Item>
                                  <Button
                                    icon={<DeleteOutlined />}
                                    onClick={() => remove(field.name)}
                                    size="small"
                                    type="text"
                                  />
                                </div>
                                <Form.Item
                                  className="mb-0"
                                  name={[field.name, 'document_ids']}
                                  rules={[{ required: true, message: t('page.settings.search_settings.labels.pinned_docs') }]}
                                >
                                  <DocTableSelect />
                                </Form.Item>
                              </div>
                            ))}
                            <Button
                              className="self-start"
                              icon={<PlusOutlined />}
                              onClick={() => add({ query: '', document_ids: [] })}
                              size="small"
                            >
                              {t('page.settings.search_settings.labels.add_pinned_rule')}
                            </Button>
                          </div>
                        )}
                      </Form.List>
                    </Form.Item>

                    <Divider />

                    <div className="color-[var(--ant-color-text)] font-medium mb-8px">
                      {t('page.settings.search_settings.labels.search_security')}
                    </div>
                    <div className="mb-16px settings-form-help">
                      {t('page.settings.search_settings.labels.search_security_hint')}
                    </div>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.apply_masking')}
                      name={['apply_masking_to_results']}
                      valuePropName="checked"
                      tooltip={t('page.settings.search_settings.labels.apply_masking_desc')}
                    >
                      <Switch size="small" />
                    </Form.Item>
                    <Form.Item
                      label={t('page.settings.search_settings.labels.blocked_query_patterns')}
                      name={['blocked_query_patterns']}
                      tooltip={t('page.settings.search_settings.labels.blocked_query_patterns_desc')}
                    >
                      <BlockedPatternsField />
                    </Form.Item>
                  </div>
                )
              }
            ]}
            onChange={setActiveTab}
          />
        </Form>
      </Spin>
      <SaveBar
        dirty={dirty || widgetDirty}
        loading={loading}
        onDiscard={handleDiscard}
        onSave={handleSaveAll}
      />
    </ListContainer>
  );
});

export default SearchSettings;
