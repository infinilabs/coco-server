import DropdownList from '@/common/src/DropdownList';
import { getLocale } from '@/store/slice/app';

// Dropdown for picking a parsing pipeline, built on the same generic
// DropdownList the model selector wraps (ModelSelect): searchable/sortable
// list with a bottom "create" action and refresh. The Form.Item value is the
// pipeline id (string); this adapter maps id <-> item object for DropdownList,
// which expects the selected object as its value.
const PipelineDropdown = ({ value, onChange, pipelineList = [], onRefresh, width }) => {
  const { t } = useTranslation();
  const locale = useAppSelector(getLocale);
  const selected = pipelineList.find(p => p.id === value);

  return (
    <DropdownList
      width={width || '100%'}
      locale={locale}
      allowClear
      data={pipelineList}
      rowKey='id'
      searchKey='name'
      renderItem={item => item.name}
      value={selected}
      onChange={item => onChange?.(item?.id)}
      placeholder={t('page.datasource.new.labels.doc_pipeline_placeholder')}
      onRefresh={onRefresh}
      actions={[
        <a
          key='create'
          onClick={() => {
            window.open('#/pipeline/new', '_blank');
          }}
        >
          {t('common.create')}
        </a>
      ]}
    />
  );
};

export default PipelineDropdown;
