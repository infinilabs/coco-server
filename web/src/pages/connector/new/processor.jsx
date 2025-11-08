import { Switch, Input } from 'antd';

export default ({ value, onChange, className }) => {

  const { t } = useTranslation();
  const [innerValue, setInnerValue] = useState(value || {});
  const handleChange = (key, val) => {
    const newValue = { ...innerValue, [key]: val };
    setInnerValue(newValue);
    onChange?.(newValue);
  };

  return (
    <div className={className}>
      <Switch
        size='small'
        onChange={(v) => handleChange('enabled', v)}
        checked={innerValue.enabled}
      />
      <div className='mt-2'>
        <div className='text-[#999]'>
          {t('page.connector.new.labels.name')}
        </div> 
        <Input
          onChange={(e) => handleChange('name', e.target.value)}
          value={innerValue.name}
        />
      </div>
    </div>
  );
};
