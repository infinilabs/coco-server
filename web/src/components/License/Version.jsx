import { Descriptions, Typography, Divider } from "antd";
import dayjs from "dayjs";
import styles from "./Version.module.less";
import { DATE_FORMAT } from ".";
import AGPL from "./AGPL";
const { Paragraph } = Typography;
import Icon from '@ant-design/icons';

export default ({ application }) => {
  const { number, build_number, build_date, build_hash } = application?.version || {};
  const { t } = useTranslation();

  return (
    <div className={styles.version}>
      <div className={styles.header}>
        <Descriptions size="small" title={`Coco Server`} column={1}>
          <Descriptions.Item
            label={t("license.labels.version")}
          >
            {number}
          </Descriptions.Item>
          <Descriptions.Item
            label={t("license.labels.build_time")}
          >
            {dayjs(build_date).format(DATE_FORMAT)}
          </Descriptions.Item>
          <Descriptions.Item
            label={t("license.labels.build_number")}
          >
            {build_number}
          </Descriptions.Item>
          <Descriptions.Item label="Hash">{build_hash}</Descriptions.Item>
        </Descriptions>
      </div>
      <div style={{ margin: '10px 0', height: 97, overflow: 'hidden' }}>
        <IconWrapper className="h-97px">
          <Icon style={{ transform: 'scale(0.2)' }} component={AGPL}/>
        </IconWrapper>
      </div>
      <Divider />
      <div className={styles.license}>
        <Paragraph>
          Copyright (C) INFINI Labs & INFINI LIMITED.
        </Paragraph>
        <Paragraph>The INFINI Console is offered under the GNU Affero General Public License v3.0 and as commercial software.</Paragraph>
        <Paragraph>
          For commercial licensing, contact us at:
          <ul>
            <li>Email: hello@infini.ltd</li>
            <li>Website: <a href={`https://coco.rs`} target="_blank">coco.rs</a></li>
          </ul>
        </Paragraph>
        <Paragraph>
          Open Source licensed under AGPL V3:
          <br />
          This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.
        </Paragraph>
        <Paragraph>This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License for more details.</Paragraph>
        <Paragraph>{`You should have received a copy of the GNU Affero General Public License along with this program. If not, see <http://www.gnu.org/licenses/>.`}</Paragraph>
      </div>
    </div>
  );
};
