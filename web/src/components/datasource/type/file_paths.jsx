import {MinusCircleOutlined, PlusOutlined} from '@ant-design/icons';
import {Button, Input} from 'antd';
import PropTypes from 'prop-types';
import React, {useEffect, useState} from 'react';
import {useTranslation} from 'react-i18next';

export const MultiFilePathInput = ({
                                     addButtonText,
                                     onChange,
                                     placeholder = 'Please input file path',
                                     value = ['']
                                   }) => {
  const [paths, setPaths] = useState(value);
  const {t} = useTranslation();

  // use useEffect to monitor external value changes and keep internal and external states synchronized
  useEffect(() => {
    // ensure value is an nonempty array
    if (Array.isArray(value) && value.length > 0) {
      setPaths(value);
    } else {
      // reset input box
      setPaths(['']);
    }
  }, [value]);

  // Encapsulates a trigger to notify the parent component that the state has changed
  const triggerChange = newPaths => {
    if (typeof onChange === 'function') {
      onChange(newPaths);
    }
  };

  // handle single input box changed
  const handlePathChange = (index, newValue) => {
    const newPaths = [...paths];
    newPaths[index] = newValue;
    setPaths(newPaths);
    triggerChange(newPaths); // notify parent component
  };

  // Add a new input box
  const addPath = () => {
    const newPaths = [...paths, ''];
    setPaths(newPaths);
    triggerChange(newPaths);
  };

  // Delete input box
  const removePath = index => {
    // prevent delete last input box
    if (paths.length <= 1) return;

    const newPaths = paths.filter((_, i) => i !== index);
    setPaths(newPaths);
    triggerChange(newPaths);
  };

  return (
    <div>
      {paths.map((path, index) => (
        <div
          className="mb-3 flex items-center"
          key={index}
        >
          <Input
            placeholder={placeholder}
            style={{width: 500}}
            value={path}
            onChange={e => handlePathChange(index, e.target.value)}
          />
          {/* optimization：show Add button on last line */}
          {index === paths.length - 1 && (
            <Button
              className="ml-5"
              icon={<PlusOutlined/>}
              type="dashed"
              onClick={addPath}
            >
              {addButtonText || t('page.datasource.file_paths_add', 'Add File Path')}
            </Button>
          )}
          {/* optimization: every input box has a delete button */}
          {paths.length > 1 && (
            <MinusCircleOutlined
              className="ml-2 cursor-pointer text-red-500"
              onClick={() => removePath(index)}
            />
          )}
        </div>
      ))}
    </div>
  );
};

// Props: validate types
MultiFilePathInput.propTypes = {
  addButtonText: PropTypes.string,
  onChange: PropTypes.func,
  placeholder: PropTypes.string,
  value: PropTypes.arrayOf(PropTypes.string)
};
