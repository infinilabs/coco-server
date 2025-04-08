import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Input, Space } from 'antd';
import React, { useState } from 'react';

export const MultiURLInput = ({ onChange, showLabel = true, value = [''] }) => {
  const [urls, setUrls] = useState(value);
  const { t } = useTranslation();

  const handleChange = (index, value) => {
    const newUrls = [...urls];
    newUrls[index] = value;
    setUrls(newUrls);
    if (typeof onChange === 'function') {
      onChange(newUrls);
    }
  };

  const addUrl = () => setUrls([...urls, '']);

  const removeUrl = index => {
    setUrls(urls.filter((_, i) => i !== index));
  };

  return (
    <div>
      {showLabel && <div className="mb-3 text-gray-400">{t('page.datasource.site_urls')}</div>}
      <div>
        {urls.map((url, index) => (
          <div
            className="mb-3 flex align-middle"
            key={index}
          >
            <Input
              style={{ width: 500 }}
              value={url}
              onChange={e => handleChange(index, e.target.value)}
            />
            {index == 0 && (
              <Button
                className="ml-5"
                icon={<PlusOutlined />}
                type="dashed"
                onClick={addUrl}
              >
                {t('page.datasource.site_urls_add')}
              </Button>
            )}
            {urls.length > 1 && index != 0 && (
              <MinusCircleOutlined
                style={{ color: 'red', cursor: 'pointer', marginLeft: 8 }}
                onClick={() => removeUrl(index)}
              />
            )}
          </div>
        ))}
      </div>
    </div>
  );
};
