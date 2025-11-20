import { Typography } from 'antd';

const { Paragraph } = Typography;

export default function EULA() {
  return (
    <div className='h-400px overflow-auto px-24px py-16px'>
      <Paragraph>Copyright (C) INFINI Labs & INFINI LIMITED.</Paragraph>
      <Paragraph>
        The Coco Server is offered under the GNU Affero General Public License v3.0 and as commercial software.
      </Paragraph>
      <Paragraph>
        For commercial licensing, contact us at:
        <ul>
          <li>Email: hello@infini.ltd</li>
          <li>
            Website:{' '}
            <a
              href='https://coco.rs'
              rel='noreferrer'
              target='_blank'
            >
              coco.rs
            </a>
          </li>
        </ul>
      </Paragraph>
      <Paragraph>
        Open Source licensed under AGPL V3:
        <br />
        This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero
        General Public License as published by the Free Software Foundation, either version 3 of the License, or (at
        your option) any later version.
      </Paragraph>
      <Paragraph>
        This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the
        implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public
        License for more details.
      </Paragraph>
      <Paragraph>{`You should have received a copy of the GNU Affero General Public License along with this program. If not, see <http://www.gnu.org/licenses/>.`}</Paragraph>
    </div>
  );
}
