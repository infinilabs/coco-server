import React, { useState } from "react";
import { Input, Button, Space } from "antd";
import { PlusOutlined, MinusCircleOutlined } from "@ant-design/icons";

export const MultiURLInput = ({value = [""], onChange, showLabel=true}) => {
  const [urls, setUrls] = useState(value);
  const { t } = useTranslation();

  const handleChange = (index, value) => {
    const newUrls = [...urls];
    newUrls[index] = value;
    setUrls(newUrls);
    if(typeof onChange === "function"){
      onChange(newUrls);
    }
  };

  const addUrl = () => setUrls([...urls, ""]);

  const removeUrl = (index) => {
    setUrls(urls.filter((_, i) => i !== index));
  };

  return (
    <div>
      {showLabel && <div className="text-gray-400 mb-3">{t('page.datasource.site_urls')}</div>}
      <div>
      {urls.map((url, index) => (
        <div key={index} className="flex align-middle mb-3">
          <Input
            value={url}
            onChange={(e) => handleChange(index, e.target.value)}
            style={{ width: 500 }}
          />
          {index==0 && (
            <Button type="dashed" className="ml-5" onClick={addUrl} icon={<PlusOutlined />}>
              {t('page.datasource.site_urls_add')}
            </Button>
          )}
          {urls.length > 1 && index != 0 && (
            <MinusCircleOutlined
              onClick={() => removeUrl(index)}
              style={{ color: "red", cursor: "pointer", marginLeft: 8 }}
            />
          )}
        </div>
      ))}
      </div>
    </div>
  );
};