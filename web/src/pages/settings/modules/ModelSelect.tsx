import DropdownList from '@/common/src/DropdownList';
import { useMemo, useState } from 'react';
import InfiniIcon from '@/components/common/icon';
import { getLocale } from '@/store/slice/app';
import { getServer } from '@/store/slice/server';

export default (props: any) => {
  const { value: propsValue, onChange, providers = [], width } = props;
  const value = propsValue ?? {};

  if (value?.provider_id && !value.id) {
    value.id = `${value.provider_id}_${value.name}`;
  }

  const grps = useMemo(() => {
    return providers.map((item: any) => {
      return `${item.id}_${item.name}`;
    });
  }, [providers]);

  const locale = useAppSelector(getLocale);
  const server = useAppSelector(getServer);

  const [sorter, setSorter] = useState([]);
  const [filters, setFilters] = useState({});
  const [groups, setGroups] = useState([]);
  const [showGroup, setShowGroup] = useState(false);

  const renderProvider = item => {
    if (!item) return null;
    return (
      <div className='flex items-center gap-4px'>
        {item.icon && (
          <IconWrapper className='h-20px w-20px'>
            <InfiniIcon
              height='1em'
              server={server}
              src={item.icon}
              width='1em'
            />
          </IconWrapper>
        )}
        <span className='font-size-1em'>{item.name}</span>
      </div>
    );
  };

  const formatData = useMemo(() => {
    const models = [];
    providers?.forEach(item => {
      (item.models || []).forEach(model => {
        models.push({
          type: `${item.id}_${item.name}`,
          provider_id: item.id,
          id: `${item.id}_${model.name}`,
          name: model.name
        });
      });
    });
    return models;
  }, [providers]);

  const filterOptions = useMemo(() => {
    return showGroup
      ? []
      : [
          {
            label: 'Type',
            key: 'type',
            list: providers.map((item: any) => ({
              key: 'type',
              value: `${item.id}_${item.name}`,
              label: renderProvider(item)
            }))
          }
        ];
  }, [showGroup, providers]);

  const groupOptions = useMemo(() => {
    return providers.map(item => ({
      label: renderProvider(item),
      key: 'type',
      value: `${item.id}_${item.name}`
    }));
  }, [showGroup, providers]);

  useEffect(() => {
    setFilters({
      type: grps
    });
  }, [grps]);

  const onSelectValueChange = (model: any) => {
    onChange?.(model);
  };

  return (
    <>
      <div className='flex items-center gap-2'>
        <DropdownList
          data={formatData}
          defaultGroupVisible={true}
          dropdownWidth={width}
          filterOptions={filterOptions}
          filters={filters}
          groupOptions={groupOptions}
          groups={groups}
          locale={locale}
          placeholder='Please select'
          renderItem={item => item.name}
          rowKey='id'
          searchKey='name'
          sorter={sorter}
          sorterOptions={[{ label: 'Name', key: 'name' }]}
          value={value}
          width={width || '100%'}
          renderLabel={item => {
            const provider = providers.find(p => p.id === item.provider_id);
            return (
              <div className='flex items-center gap-2px'>
                {provider && (
                  <>
                    <span>{renderProvider(provider)}</span>
                    <span>/</span>
                  </>
                )}
                <span>{item.name}</span>
              </div>
            );
          }}
          onChange={onSelectValueChange}
          onFiltersChange={setFilters}
          onSorterChange={setSorter}
          onGroupsChange={v => {
            setGroups(v);
          }}
          onGroupVisibleChange={visible => {
            setShowGroup(visible);
            if (visible) {
              setFilters({});
              setGroups([{ key: 'type', value: grps[0] }]);
            } else {
              setGroups([]);
              setFilters({ type: grps });
            }
          }}
        />
      </div>
    </>
  );
};
