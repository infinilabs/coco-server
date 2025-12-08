import { Descriptions } from 'antd';
import dayjs from 'dayjs';
import PropTypes from 'prop-types';
import Icon from '@ant-design/icons';
import { getDarkMode } from "@/store/slice/theme";

import { DATE_FORMAT } from '.';
import styles from './Version.module.less';

Version.propTypes = {
  application: PropTypes.shape({
    version: PropTypes.shape({
      number: PropTypes.string,
      lucene_version: PropTypes.string,
      build_snapshot: PropTypes.bool,
      build_date: PropTypes.string,
      build_hash: PropTypes.string
    })
  })
};

export default function Version({ application }) {
  const { number, build_number, build_date, build_hash } = application?.version || {};
  const { t } = useTranslation();

  const darkMode = useAppSelector(getDarkMode);

  return (
    <div className={styles.version}>
      <div className={styles.header}>
        <Descriptions
          column={1}
          size='small'
          title={
            <div className='mt-24px flex items-center' style={darkMode ? { filter: `drop-shadow(0 0 6px rgba(255, 255, 255, 1))`} : {}}>
              <SvgIcon
                className='h-85px w-300px'
                localIcon='logo'
              />
            </div>
          }
        >
          <Descriptions.Item label={t('license.labels.version')}>{number}</Descriptions.Item>
          <Descriptions.Item label={t('license.labels.build_time')}>
            {dayjs(build_date).format(DATE_FORMAT)}
          </Descriptions.Item>
          <Descriptions.Item label={t('license.labels.build_number')}>{build_number}</Descriptions.Item>
          <Descriptions.Item label='Hash'>{build_hash}</Descriptions.Item>
        </Descriptions>
      </div>
      <div style={{ margin: '10px 0', height: 97, overflow: 'hidden' }}>
        <IconWrapper className='h-97px'>
          <Icon
            component={AGPL}
            style={{ transform: 'scale(0.2)' }}
          />
        </IconWrapper>
      </div>
    </div>
  );
}
