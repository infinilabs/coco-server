import DropdownList from "@/common/src/DropdownList";
import { useMemo, useState } from "react";
import "./ModelSelect.scss";

export default (props) => {

    const { value, onChange, providers = [], width } = props;
    if(value && !value.id){
      value.id = value.provider_id + "_" + value.name;
    }

    const grps = providers.map((item: any) => {
      return item.name;
    })

    
    const [sorter, setSorter] = useState([])
    const [filters, setFilters] = useState({ type: grps })
    const [groups, setGroups] = useState([])
    const [showGroup, setShowGroup] = useState(false)

    const formatData = useMemo(() => {
      const models = [];
      providers?.forEach((item) => {
        (item.models || []).forEach((model) => {
          models.push({
            type: item.name,
            provider_id: item.id,
            id: item.id + "_" + model.name,
            name: model.name,
          })
        })
      });
      return models;
    }, [providers])

    const filterOptions = useMemo(() => {
      return showGroup ? [] : [{ 
        label: "Type", 
        key: "type", 
        list: grps.map((grp: any)=>({label: grp, value: grp})),
    }]
    }, [showGroup])

    const onSelectValueChange = (value: any) => {
      const newValue = {
        name: value?.name,
        provider_id: value?.provider_id,
      }
      onChange?.(newValue);
    }

    return (
      <div className="flex gap-2 items-center">
        <DropdownList
          className="model-select"
          value={value}
          onChange={onSelectValueChange}
          placeholder="Please select"
          rowKey="id"
          data={formatData}
          renderItem={(item) => (
              item.name
          )}
          width={width || "100%"}
          dropdownWidth={width}
          renderLabel={(item) => item.name}
          searchKey="name"
          sorter={sorter}
          onSorterChange={setSorter}
          sorterOptions={[
            { label: "Name", key: "name" },
          ]}
          filters={filters}
          onFiltersChange={setFilters}
          filterOptions={filterOptions}
          groups={groups}
          onGroupsChange={(v)=>{setGroups(v)}}
          groupOptions={grps.map(grp=>({
            label: grp,
            key: "type",
            value: grp
          }))}
          onGroupVisibleChange={(visible) => {
            setShowGroup(visible)
            if (visible) {
              setFilters({})
              setGroups([{ key: 'type', value: grps[0]}])
            } else {
              setGroups([])
              setFilters({ type: grps})
            }
          }}
        />
      </div>
    )
}