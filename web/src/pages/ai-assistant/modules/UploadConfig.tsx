import { Input, InputNumber, Select, Space, Switch } from 'antd';

interface UploadConfigProps {
  readonly value?: any;
  readonly onChange?: (value: string) => void;
}

function formatBytes(bytes) {
  if (bytes === 0 || isNaN(bytes) || bytes === null) {
    return {
      value: 1,
      unit: 1024 * 1024
    };
  }

  const base = 1024;
  const units = [1, 1024, 1024 * 1024, 1024 * 1024 * 1024];

  let unitIndex = 0;
  while (bytes >= base && unitIndex < units.length - 1) {
    bytes /= base;
    unitIndex++;
  }

  const formattedValue = Number.parseFloat(bytes.toFixed(0));

  return {
    value: formattedValue,
    unit: units[unitIndex]
  };
}

export const UploadConfig = (props: UploadConfigProps) => {
  const { t } = useTranslation();
  const { value = {}, onChange } = props;

  const [maxFileSize, setMaxFileSize] = useState({ value: 1, unit: 1024 * 1024 });
  const formatRef = useRef(true);

  useEffect(() => {
    if (formatRef.current && value.max_file_size_in_bytes) {
      formatRef.current = false;
      setMaxFileSize(formatBytes(value.max_file_size_in_bytes));
    }
  }, [value.max_file_size_in_bytes]);

  const onEnabledChange = (enabled: boolean) => {
    onChange?.({
      ...value,
      allowed_file_extensions: Array.isArray(value.allowed_file_extensions) ? value.allowed_file_extensions : ['*'],
      max_file_size_in_bytes: value.max_file_size_in_bytes || 1024 * 1024,
      max_file_count: value.max_file_count || 6,
      enabled
    });
  };

  return (
    <Space
      className='mt-[5px] w-600px'
      direction='vertical'
    >
      <div>
        <Switch
          size='small'
          value={value.enabled}
          onChange={onEnabledChange}
        />
      </div>
      <div>
        <Space
          className='w-full'
          direction='vertical'
        >
          <p className='mt-10px text-[#999]'>{t('page.assistant.labels.allowed_file_extensions')}</p>
          <Input
            className='w-full'
            value={Array.isArray(value.allowed_file_extensions) ? value.allowed_file_extensions.join(',') : '*'}
            onChange={e => {
              const newValue = e.target.value.replace(/[.\s]/g, '');
              onChange?.({
                ...value,
                allowed_file_extensions: newValue ? newValue.split(',') : []
              });
            }}
          />
        </Space>
      </div>
      <div>
        <Space direction='vertical'>
          <p className='mt-10px text-[#999]'>{t('page.assistant.labels.max_file_size_in_bytes')}</p>
          <InputNumber
            className='w-148px'
            min={1}
            value={maxFileSize.value}
            addonAfter={
              <Select
                className='w-auto'
                popupMatchSelectWidth={60}
                value={maxFileSize.unit}
                onChange={v => {
                  setMaxFileSize({
                    ...maxFileSize,
                    unit: v
                  });
                  onChange?.({
                    ...value,
                    max_file_size_in_bytes: maxFileSize.value * v
                  });
                }}
              >
                <Select.Option value={1}>B</Select.Option>
                <Select.Option value={1024}>KB</Select.Option>
                <Select.Option value={1024 * 1024}>MB</Select.Option>
                <Select.Option value={1024 * 1024 * 1024}>GB</Select.Option>
              </Select>
            }
            onChange={v => {
              setMaxFileSize({
                ...maxFileSize,
                value: v || 1
              });
              onChange?.({
                ...value,
                max_file_size_in_bytes: (v || 1) * maxFileSize.unit
              });
            }}
          />
        </Space>
      </div>
      <div>
        <Space direction='vertical'>
          <p className='mt-10px text-[#999]'>{t('page.assistant.labels.max_file_count')}</p>
          <InputNumber
            max={100}
            min={1}
            value={value.max_file_count || 6}
            onChange={v => {
              onChange?.({
                ...value,
                max_file_count: v
              });
            }}
          />
        </Space>
      </div>
      {/* <div>
        <Space direction="vertical">
          <p className="text-[#999] mt-10px">
            {t("page.assistant.labels.caller_model")}
          </p>
        </Space>
      </div> */}
    </Space>
  );
};
