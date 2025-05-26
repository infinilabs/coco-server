import { PlusOutlined } from '@ant-design/icons';
import type { InputRef } from 'antd';
import { Input, Tag, theme } from 'antd';
import React, { useEffect, useRef, useState } from 'react';

interface TagsProps {
  readonly onChange?: (newTags: string[]) => void;
  readonly value?: string[];
}

export const Tags: React.FC<TagsProps> = ({ onChange, value }) => {
  const { token } = theme.useToken();
  const [tags, setTags] = useState<string[]>(value || []);
  const [inputVisible, setInputVisible] = useState(false);
  const [inputValue, setInputValue] = useState('');
  const inputRef = useRef<InputRef>(null);

  useEffect(() => {
    setTags(value || [])
  }, [value])
  
  useEffect(() => {
    if (inputVisible) {
      inputRef.current?.focus();
    }
  }, [inputVisible]);

  const handleClose = (removedTag: string) => {
    const newTags = tags.filter(tag => tag !== removedTag);
    setTags(newTags);
    onChange?.(newTags);
  };

  const showInput = () => {
    setInputVisible(true);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setInputValue(e.target.value);
  };

  const handleInputConfirm = () => {
    if (inputValue.trim() && !tags.includes(inputValue.trim())) {
      const newTags = [...tags, inputValue.trim()];
      setTags(newTags);
      onChange?.(newTags);
    }
    setInputVisible(false);
    setInputValue('');
  };

  const tagPlusStyle: React.CSSProperties = {
    background: token.colorBgContainer,
    borderStyle: 'dashed',
    cursor: 'pointer',
  };

  return (
    <div >
      {tags.map(tag => (
        <Tag
          closable
          key={tag}
          onClose={e => {
            e.preventDefault();
            handleClose(tag);
          }}
        >
          {tag}
        </Tag>
      ))}
      {inputVisible ? (
        <Input
          ref={inputRef}
          size="small"
          style={{ width: 100 }}
          type="text"
          value={inputValue}
          onBlur={handleInputConfirm}
          onChange={handleInputChange}
          onPressEnter={handleInputConfirm}
        />
      ) : (
        <Tag
          style={tagPlusStyle}
          onClick={showInput}
        >
          <PlusOutlined /> New Tag
        </Tag>
      )}
    </div>
  );
};
