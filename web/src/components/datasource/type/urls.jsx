import React, { useState } from "react";
import { Input, Button, Space } from "antd";
import { PlusOutlined, MinusCircleOutlined } from "@ant-design/icons";

export const MultiURLInput = ({value = [""], onChange}) => {
  const [urls, setUrls] = useState(value);

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
      <div className="text-gray-400">Site URLs</div>
      <div>
      {urls.map((url, index) => (
        <div key={index} className="flex align-middle my-3">
          <Input
            value={url}
            onChange={(e) => handleChange(index, e.target.value)}
            placeholder="Enter URL"
            style={{ width: 500 }}
          />
          {index==0 && (
            <Button type="dashed" className="ml-5" onClick={addUrl} icon={<PlusOutlined />}>
            Add URL
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