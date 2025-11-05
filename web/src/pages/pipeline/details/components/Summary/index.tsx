import { FlowGraph } from '@ant-design/graphs';
import { Flex, Table } from 'antd';

const Summary = () => {
  const [data, setData] = useState(undefined);

  useMount(() => {
    fetch('https://assets.antv.antgroup.com/g6/flow-analysis.json')
      .then(res => res.json())
      .then(setData);
  });

  const columns = [
    {
      dataIndex: 'step',
      title: 'Step'
    },
    {
      dataIndex: 'type',
      title: 'Type'
    },
    {
      dataIndex: 'type',
      title: 'Type'
    },
    {
      dataIndex: 'message',
      title: 'Message'
    },
    {
      dataIndex: 'timestamp',
      title: 'Timestamp'
    }
  ];

  return (
    <Flex
      vertical
      gap={32}
    >
      <FlowGraph
        autoFit="view"
        data={data}
        height={350}
        labelField={d => d.value.title}
      />

      <Flex
        vertical
        gap={16}
      >
        <b>运行提醒</b>

        <Table
          bordered
          columns={columns}
        />
      </Flex>
    </Flex>
  );
};

export default Summary;
