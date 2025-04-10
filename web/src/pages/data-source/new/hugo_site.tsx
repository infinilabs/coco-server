import { Form } from 'antd';

import { MultiURLInput } from '@/components/datasource/type/urls';

export default () => {
  const { t } = useTranslation();
  return (
    <Form.Item
      label={t('page.datasource.new.labels.site_urls')}
      name="urls"
      rules={[{ message: 'Please input site url!', required: true }]}
    >
      <MultiURLInput showLabel={false} />
    </Form.Item>
  );
};
