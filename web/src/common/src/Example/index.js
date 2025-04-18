import { Typography } from 'antd';
import styles from './index.module.less';

const { Title, Paragraph } = Typography;

const Example = () => {
  return (
    <div className={styles.example}>
      <Typography>
        <Title>Introduction</Title>
        <Paragraph>This is an example!</Paragraph>
      </Typography>
    </div>
  );
};

export default Example;
