import { Switch } from 'antd';
import { useState } from 'react';

export const IndexingScope = ({
  onChange,
  value = {
    include_private_book: true,
    include_private_doc: true,
    indexing_books: true,
    indexing_docs: true,
    indexing_groups: true,
    indexing_users: false
  }
}) => {
  const [innerV, setValue] = useState(value || {});
  const onPartChange = (key, v) => {
    setValue(oldV => {
      const newV = {
        ...oldV,
        [key]: v
      };
      if (typeof onChange === 'function') {
        onChange(newV);
      }
      return newV;
    });
  };
  return (
    <div>
      <div className="max-w-200px flex justify-between pb-10px pt-5px items-center">
        <span className="text-gray-400">Indexing Books</span>
        <Switch
          size="small"
          value={innerV.indexing_books}
          onChange={v => {
            onPartChange('indexing_books', v);
          }}
        />
      </div>
      <div className="max-w-200px flex justify-between py-10px items-center">
        <span className="text-gray-400">Indexing Docs</span>
        <Switch
          size="small"
          value={innerV.indexing_docs}
          onChange={v => {
            onPartChange('indexing_docs', v);
          }}
        />
      </div>
      <div className="max-w-200px flex justify-between py-10px items-center">
        <span className="text-gray-400">Indexing Groups</span>
        <Switch
          size="small"
          value={innerV.indexing_groups}
          onChange={v => {
            onPartChange('indexing_groups', v);
          }}
        />
      </div>
      <div className="max-w-200px flex justify-between py-10px items-center">
        <span className="text-gray-400">Indexing Users</span>
        <Switch
          size="small"
          value={innerV.indexing_users}
          onChange={v => {
            onPartChange('indexing_users', v);
          }}
        />
      </div>
      <div className="max-w-200px flex justify-between py-10px items-center">
        <span className="text-gray-400">Include Private Book</span>
        <Switch
          size="small"
          value={innerV.include_private_book}
          onChange={v => {
            onPartChange('include_private_book', v);
          }}
        />
      </div>
      <div className="max-w-200px flex justify-between py-10px items-center">
        <span className="text-gray-400">Include Private Doc</span>
        <Switch
          size="small"
          value={innerV.include_private_doc}
          onChange={v => {
            onPartChange('include_private_doc', v);
          }}
        />
      </div>
    </div>
  );
};
