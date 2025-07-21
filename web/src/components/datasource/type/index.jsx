import Icon, { UploadOutlined } from '@ant-design/icons';
import { Button, Form, Input, Upload } from 'antd';
import React, { useCallback, useState } from 'react';

import { BucketSVG, GoogleDriveSVG, HugoSVG, NotionSVG, SearchSVG, YuqueSVG } from '../../icons';
import { IndexingScope } from '../indexing_scope';

import { MultiURLInput } from './urls';

export const Types = {
  GoogleDrive: 'google_drive',
  HugoSite: 'hugo_site',
  LocalFS: 'local_fs',
  Notion: 'notion',
  ObjectStorage: 'object_storage',
  RSS: 'rss',
  Search: 'search',
  Yuque: 'yuque'
};

export const TypeList = ({
  mode = 'edit',
  onChange = () => {},
  onTestClick,
  value = {
    config: {},
    id: Types.GoogleDrive
  }
}) => {
  const { t } = useTranslation();
  const [v, setValue] = useState(value);
  const onItemClick = type => {
    setValue(oldV => {
      const newV = {
        ...oldV,
        id: type
      };
      if (typeof onChange === 'function') {
        onChange(newV);
      }
      return newV;
    });
  };
  const onInnerTestClick = useCallback(() => {
    if (typeof onTestClick === 'function') {
      onTestClick(v);
    }
  }, [v]);
  const onTokenChange = tv => {
    setValue(oldV => {
      const newV = {
        ...oldV
      };
      newV.config.token = tv.target.value;
      if (typeof onChange === 'function') {
        onChange(newV);
      }
      return newV;
    });
  };
  const onIndexingScopeChange = scope => {
    setValue(oldV => {
      const newV = {
        ...oldV
      };
      newV.config = {
        ...oldV.config,
        ...scope
      };
      if (typeof onChange === 'function') {
        onChange(newV);
      }
      return newV;
    });
  };
  const onSiteURLsChange = urls => {
    setValue(oldV => {
      const newV = {
        ...oldV,
        config: {
          ...oldV.config,
          urls
        }
      };
      if (typeof onChange === 'function') {
        onChange(newV);
      }
      return newV;
    });
  };
  const scope = {
    include_private_book: Boolean(v.config.include_private_book),
    include_private_doc: Boolean(v.config.include_private_doc),
    indexing_books: Boolean(v.config.indexing_books),
    indexing_docs: Boolean(v.config.indexing_docs),
    indexing_groups: Boolean(v.config.indexing_groups),
    indexing_users: Boolean(v.config.indexing_users)
  };

  const onCredentialChange = credential => {
    setValue(oldV => {
      const newV = {
        ...oldV,
        config: {
          ...oldV.config,
          credential
        }
      };
      if (typeof onChange === 'function') {
        onChange(newV);
      }
      return newV;
    });
  };
  return (
    <div>
      {mode === 'edit' && (
        <div className="flex gap-10px">
          <TypeComponent
            icon={GoogleDriveSVG}
            name={Types.GoogleDrive}
            selected={v.id === Types.GoogleDrive}
            text="Google Drive"
            onChange={onItemClick}
          />
          <TypeComponent
            icon={HugoSVG}
            name={Types.HugoSite}
            selected={v.id === Types.HugoSite}
            text="HUGO Site"
            onChange={onItemClick}
          />
          <TypeComponent
            icon={YuqueSVG}
            name={Types.Yuque}
            selected={v.id === Types.Yuque}
            text="Yuque"
            onChange={onItemClick}
          />
          <TypeComponent
            icon={NotionSVG}
            name={Types.Notion}
            selected={v.id === Types.Notion}
            text="Notion"
            onChange={onItemClick}
          />
          {/* <TypeComponent onChange={onItemClick} icon={BucketSVG} text="Object Storage" selected={v===Types.ObjectStorage} name={Types.ObjectStorage}/>
      <TypeComponent onChange={onItemClick} icon={SearchSVG} text="Search" selected={v===Types.Search} name={Types.Search}/> */}
        </div>
      )}
      {(v.id === Types.Notion || v.id === Types.Yuque) && (
        <div className={mode === 'edit' ? 'my-20px' : ''}>
          <div className="pb-8px text-gray-400">Token</div>
          <div className="flex gap-5px">
            <Input.Password
              className="max-w-500px"
              value={v.config.token}
              onChange={onTokenChange}
            />
            <Button onClick={onInnerTestClick}>{t('common.testConnection')}</Button>
          </div>
        </div>
      )}
      {v.id === Types.Yuque && (
        <IndexingScope
          value={scope}
          onChange={onIndexingScopeChange}
        />
      )}
      {v.id === Types.HugoSite && (
        <div className={mode === 'edit' ? 'my-20px' : ''}>
          <MultiURLInput
            value={v.config?.urls || ['']}
            onChange={onSiteURLsChange}
          />
        </div>
      )}
      {v.id === Types.GoogleDrive && (
        <div className={mode === 'edit' ? 'my-20px' : ''}>
          <Button>
            <a href={`${location.origin}/connector/google_drive/connect`}>Connect</a>
          </Button>
          {/* <FileUploader onChange={onCredentialChange}/>
      <Button onClick={onInnerTestClick}>{t('common.testConnection')}</Button> */}
          {/* </div> */}
        </div>
      )}
    </div>
  );
};

const TypeComponent = ({ icon, name, onChange = () => {}, selected = false, text }) => {
  return (
    <div
      className={`border flex items-center px-10px py-5px rounded-md min-w-120px justify-center hover:border-blue-500 hover:text-blue-500 cursor-pointer${
        selected ? ' border-blue-500 text-blue-500' : ''
      }`}
      onClick={() => {
        onChange(name);
      }}
    >
      <Icon component={icon} />
      <span className="ml-2">{text}</span>
    </div>
  );
};

const Credential = ({ onChange, value }) => {
  const { t } = useTranslation();
  const [v, setValue] = useState(value);
  const onCredentialChange = (e, key) => {
    setValue(oldV => {
      const newV = {
        ...oldV,
        [key]: e.target.value
      };
      if (typeof onChange === 'function') {
        onChange(newV);
      }
      return newV;
    });
  };
  const fields = [
    { key: 'client_id', label: t('page.datasource.new.labels.client_id') },
    { key: 'client_secret', label: t('page.datasource.new.labels.client_secret') },
    { key: 'redirect_uri', label: t('page.datasource.new.labels.redirect_uri') },
    { key: 'endpoint', label: 'Endpoint' }
  ];

  return (
    <div className="grid grid-cols-[100px_1fr] items-center gap-x-4 gap-y-2">
      {fields.map(({ key, label }) => (
        <React.Fragment key={key}>
          <div className="text-left text-gray-400">{label}</div>
          <Input
            className="max-w-[485px] w-full"
            value={v[key]}
            onChange={e => onCredentialChange(e, key)}
          />
        </React.Fragment>
      ))}
    </div>
  );
};

const FileUploader = ({ onChange }) => {
  const [fileList, setFileList] = useState([]);
  // const [credential, setCredential] = useState({});

  const props = {
    beforeUpload: file => {
      const isJson = file.type === 'application/json';

      if (!isJson) {
        console.error('Only JSON files are allowed!');
        return Upload.LIST_IGNORE; // Prevents adding non-JSON files
      }

      const reader = new FileReader();
      reader.onload = e => {
        try {
          const fileContent = JSON.parse(e.target.result);
          if (typeof onChange === 'function') {
            onChange(fileContent);
          }
        } catch (error) {
          console.error('Invalid JSON file!', error);
        }
      };

      reader.readAsText(file); // Read the file as text

      return false; // Prevents automatic upload
    },
    fileList,
    onRemove: () => {
      setCredential({});
    }
  };
  return (
    <Upload
      {...props}
      accept=".json"
      maxCount={1}
    >
      <Button icon={<UploadOutlined />}>Click To Upload File</Button>
    </Upload>
  );
};
