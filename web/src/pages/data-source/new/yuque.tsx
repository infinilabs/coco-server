import { Button, Form, Input, Switch } from 'antd';

import { IndexingScope } from '@/components/datasource/indexing_scope';

export const TokenInput = ({ onChange = () => {}, value = '' }) => {
  const { t } = useTranslation();
  return (
    <div>
      <Input.Password
        className="max-w-500px"
        value={value}
        onChange={onChange}
      />
      {/* <Button className="ml-10px">{t('common.testConnection')}</Button> */}
    </div>
  );
};

export default () => {
  const { t } = useTranslation();
  return (
    <>
      <Form.Item
        label="Token"
        name="token"
      >
        <TokenInput />
      </Form.Item>
      <Form.Item
        initialValue={true}
        label={t('page.datasource.new.labels.indexing_scope')}
        name="indexing_scope"
      >
        <IndexingScope />
      </Form.Item>
    </>
  );
};
