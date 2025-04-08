import { Form } from 'antd';

import { TokenInput } from './yuque';

export default () => {
  return (
    <Form.Item
      label="Token"
      name="token"
    >
      <TokenInput />
    </Form.Item>
  );
};
