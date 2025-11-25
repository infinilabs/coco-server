import { Checkbox, Flex, Form, Switch } from 'antd';
import { capitalize } from 'lodash';

export const ToolsConfig = () => {
  const { t } = useTranslation();

  return (
    <>
      <Form.Item
        className='mb-0 mt-3'
        label={t('page.assistant.labels.built_in_large_model_tool')}
        name={['tools', 'enabled']}
      >
        <Switch size='small' />
      </Form.Item>

      <Flex
        className='[&>div]:w-1/2'
        wrap='wrap'
      >
        {['calculator', 'wikipedia', 'duckduckgo', 'scraper'].map(item => (
          <Form.Item
            className='mb-0'
            key={item}
            label={capitalize(item)}
            name={['tools', 'builtin', item]}
          >
            <Checkbox />
          </Form.Item>
        ))}
      </Flex>
    </>
  );
};
