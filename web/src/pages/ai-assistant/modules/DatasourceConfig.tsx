import { Flex, Form, Input, Select, Space, Switch } from 'antd';

interface DatasourceConfigProps {
  readonly value?: any;
  readonly onChange?: (value: string) => void;
  readonly options: any[];
  readonly loading: boolean;
}

export const DatasourceConfig = (props: DatasourceConfigProps) => {
  const { t } = useTranslation();
  const { loading, value = {}, onChange } = props;

  const [showFilter, setShowFilter] = useState(Boolean(value.filter));

  const onFilterToggle = () => {
    setShowFilter(!showFilter);
  };

  const filterPlaceHolder = `{
  "term": {
     "name": "test"
  }
}`;

  useEffect(() => {
    if (value.visible) return;

    onChange?.({
      ...value,
      enabled_by_default: true
    });
  }, [value.visible]);

  return (
    <Space
      className='w-full'
      direction='vertical'
    >
      <div className='mb-3 font-bold'>{t('page.assistant.labels.document_retrieval')}</div>

      <Form.Item
        label={t('page.assistant.labels.datasource')}
        layout='vertical'
        name={['datasource', 'ids']}
      >
        <Select
          allowClear
          loading={loading}
          mode='multiple'
          options={props.options}
          value={value.ids}
        />
      </Form.Item>

      <div
        className='mt-10px flex cursor-pointer items-center text-blue-500'
        onClick={onFilterToggle}
      >
        <span>{t('page.assistant.labels.filter')}</span>

        <SvgIcon
          className='text-20px'
          icon={showFilter ? 'mdi:chevron-up' : 'mdi:chevron-down'}
        />
      </div>

      {showFilter && (
        <Form.Item
          className='mb-0'
          name={['datasource', 'filter']}
        >
          <Input.TextArea
            placeholder={filterPlaceHolder}
            style={{ height: 150 }}
          />
        </Form.Item>
      )}

      <div className='mt-3 -mb-1'>{t('page.assistant.labels.feature_visibility')}</div>

      <Flex className='[&>div]:(flex-1 m-0!)'>
        <Form.Item
          label={t('page.assistant.labels.show_in_chat')}
          name={['datasource', 'visible']}
        >
          <Switch size='small' />
        </Form.Item>

        <Form.Item
          label={t('page.assistant.labels.enabled_by_default')}
          name={['datasource', 'enabled_by_default']}
        >
          <Switch
            disabled={!value.visible}
            size='small'
          />
        </Form.Item>
      </Flex>
    </Space>
  );
};
