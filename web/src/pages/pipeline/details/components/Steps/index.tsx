import { Collapse } from 'antd';

const Steps = () => {
  return (
    <Collapse
      className="bg-black/2 flex-none! dark:bg-white/4"
      defaultActiveKey={['1']}
      expandIconPosition="end"
      items={[
        {
          children: <p>消息清洗</p>,
          key: '1',
          label: '消息清洗'
        },
        {
          children: <p>FAQ 提取</p>,
          key: '2',
          label: 'FAQ 提取'
        },
        {
          children: <p>Embedding 生成</p>,
          key: '3',
          label: 'Embedding 生成'
        },
        {
          children: <p>保存结果</p>,
          key: '4',
          label: '保存结果'
        }
      ]}
    />
  );
};

export default Steps;
