import { Button } from 'antd';
import styles from './Apply.less';

const Apply = props => {
  const { currentLocales, onApply, onCancel } = props;

  return (
    <div className={styles.apply}>
      <Button type="link" onClick={onCancel}>
        {currentLocales[`datepicker.cancel`]}
      </Button>
      <Button className={styles.applyBtn} type="primary" onClick={onApply}>
        {currentLocales[`datepicker.apply`]}
      </Button>
    </div>
  );
};

export default Apply;
