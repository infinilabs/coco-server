import { DeleteOutlined } from '@ant-design/icons';
import { Button, Col, Input, Row } from 'antd';
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { IconSelector } from './icon_selector';

export const AssetsIcons = ({ iconsMeta = [], onChange, value = {} }) => {
  const { t } = useTranslation();
  const initialIcons = Object.keys(value).map(k => {
    return {
      icon: value[k],
      key: new Date().getTime(),
      type: k
    };
  });
  if (!initialIcons.length) {
    initialIcons.push({
      icon: '',
      key: new Date().getTime(),
      type: ''
    });
  }
  const transformIcons = icons => {
    return (icons || []).reduce((acc, icon) => {
      if (!icon.type || !icon.icon) {
        return acc;
      }
      acc[icon.type] = icon.icon;
      return acc;
    }, {});
  };
  const [icons, setIcons] = useState(initialIcons);
  const onIconChange = (oldKey, v) => {
    setIcons(oldIcons => {
      const idx = oldIcons.findIndex(icon => icon.key === oldKey);
      if (idx > -1) {
        oldIcons[idx] = v;
      }
      typeof onChange === 'function' && onChange(transformIcons(oldIcons));
      return [...oldIcons];
    });
  };
  const onIconRemove = v => {
    setIcons(oldIcons => {
      oldIcons = oldIcons.filter(icon => {
        return icon.key !== v.key;
      });
      typeof onChange === 'function' && onChange(transformIcons(oldIcons));
      return [...oldIcons];
    });
  };

  const onAddIcon = () => {
    setIcons(oldIcons => {
      return [
        ...oldIcons,
        {
          icon: '',
          key: new Date().getTime(),
          type: ''
        }
      ];
    });
  };
  return (
    <div className="assets-icons mt-3px">
      <Row
        className="w-100%"
        gutter={{ lg: 32, md: 24, sm: 16, xs: 8 }}
      >
        <Col
          key="icon"
          span={10}
        >
          {t('page.connector.new.labels.asset_icon')}
        </Col>
        <Col
          key="type"
          span={5}
        >
          {t('page.connector.new.labels.asset_type')}
        </Col>
        <Col
          key="oper"
          span={8}
        />
      </Row>
      {icons.map(item => {
        return (
          <AssetsIcon
            icons={iconsMeta}
            key={item.key}
            value={item}
            onChange={onIconChange}
            onRemove={onIconRemove}
          />
        );
      })}
      <Button
        className="mt-10px"
        type="primary"
        onClick={onAddIcon}
      >
        {t('common.add')}
      </Button>
    </div>
  );
};

const AssetsIcon = ({ icons = [], onChange, onRemove, value = {} }) => {
  const [innerValue, setInnerValue] = useState(value);
  const onDeleteClick = () => {
    typeof onRemove === 'function' && onRemove(value);
  };

  const onTypeChange = e => {
    setInnerValue(v => {
      const newV = {
        ...v,
        type: e.target.value
      };
      onChange?.(value.key, newV);
      return newV;
    });
  };
  const onIconChange = (icon, option) => {
    setInnerValue(v => {
      const newV = {
        ...v,
        icon
      };
      if (!newV.type) {
        newV.type = option?.item?.name || '';
      }
      onChange?.(value.key, newV);
      return newV;
    });
  };

  return (
    <Row
      className="my-5px w-100%"
      gutter={{ lg: 32, md: 24, sm: 16, xs: 8 }}
    >
      <Col span={10}>
        <IconSelector
          icons={icons}
          type="file"
          value={innerValue.icon || ''}
          onChange={onIconChange}
        />
      </Col>
      <Col span={5}>
        <Input
          value={innerValue.type || ''}
          onChange={onTypeChange}
        />
      </Col>
      <Col
        className="flex items-center"
        span={8}
      >
        <div
          className="cursor-pointer"
          onClick={onDeleteClick}
        >
          <DeleteOutlined />
        </div>
      </Col>
    </Row>
  );
};
