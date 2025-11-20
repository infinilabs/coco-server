import { Descriptions } from 'antd';
import dayjs from 'dayjs';
import PropTypes from 'prop-types';

import { DATE_FORMAT } from '.';

LicenseInfo.propTypes = {
  license: PropTypes.shape({
    license_type: PropTypes.string,
    issue_to: PropTypes.string,
    issue_at: PropTypes.string,
    expire_at: PropTypes.string,
    max_nodes: PropTypes.oneOfType([PropTypes.string, PropTypes.number])
  })
};

export default function LicenseInfo({ license }) {
  const { license_type, issue_to = '-', issue_at, expire_at, max_nodes = '-' } = license || {};

  const { t } = useTranslation();

  return (
    <Descriptions
      column={1}
      size='small'
      title=''
    >
      <Descriptions.Item label={t('license.labels.license_type')}>{license_type}</Descriptions.Item>
      <Descriptions.Item label={t('license.labels.max_nodes')}>{max_nodes}</Descriptions.Item>
      <Descriptions.Item label={t('license.labels.issue_to')}>{issue_to}</Descriptions.Item>
      <Descriptions.Item label={t('license.labels.issue_at')}>
        {issue_at ? dayjs(issue_at).format(DATE_FORMAT) : '-'}
      </Descriptions.Item>
      <Descriptions.Item label={t('license.labels.expire_at')}>
        {expire_at ? dayjs(expire_at).format(DATE_FORMAT) : '-'}
      </Descriptions.Item>
    </Descriptions>
  );
}
