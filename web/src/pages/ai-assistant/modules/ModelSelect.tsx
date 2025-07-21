import DropdownList from "@/common/src/DropdownList";
import { useMemo, useState } from "react";
import ModelSettings from "./ModelSettings";
import InfiniIcon from '@/components/common/icon';

const DefaultModelSettings = {
  settings: {
    temperature: 0.7,
    top_p: 0.9,
    presence_penalty: 0,
    frequency_penalty: 0,
    max_tokens: 4000,
  },
  prompt: {
    template: `
  You are a helpful AI assistant.
  You will be given a conversation below and a follow-up question.

  {{.context}}

  The user has provided the following query:
  {{.query}}

  Ensure your response is thoughtful, accurate, and well-structured.
  For complex answers, format your response using clear and well-organized **Markdown** to improve readability.
  `,
  }
}

export default (props) => {

    const { value: propsValue, onChange, providers = [], width } = props;
    const value = propsValue ?? { ...DefaultModelSettings };

    if(value?.provider_id && !value.id){
      value.id = value.provider_id + "_" + value.name;
    }
    if(!value.settings){
      value.settings = DefaultModelSettings.settings;
    }
    if(!value.prompt){
      value.prompt = DefaultModelSettings.prompt;
    }

    const grps = useMemo(() => {
      return providers.map((item: any) => {
        return item.id + "_" + item.name;
      })
    }, [providers]) 
    
    const [sorter, setSorter] = useState([])
    const [filters, setFilters] = useState({})
    const [groups, setGroups] = useState([])
    const [showGroup, setShowGroup] = useState(false)

    const renderProvider = (item) => {
      if (!item) return null;
      return (
        <div className="flex items-center gap-4px">
          {
            item.icon && (
              <IconWrapper className="w-20px h-20px">
                <InfiniIcon src={item.icon} height="1em" width="1em" />
              </IconWrapper>
            )
          }
          <span className="font-size-1em">{item.name}</span>
        </div>
      )
    }

    const formatData = useMemo(() => {
      const models = [];
      providers?.forEach((item) => {
        (item.models || []).forEach((model) => {
          models.push({
            type: item.id + "_" + item.name,
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
        list: providers.map((item: any) => ({
          key: "type",
          value: item.id + "_" + item.name,
          label: renderProvider(item)
        }))
      }]
    }, [showGroup, providers])

    const groupOptions = useMemo(() => {
      return providers.map(item =>({
        label: renderProvider(item),
        key: "type",
        value: item.id + "_" + item.name
      }))
    }, [showGroup, providers])

    useEffect(() => {
      setFilters({
        type: grps
      })
    }, [grps])

    const onSelectValueChange = (model: any) => {
      onChange?.(model);
    }

    const onSettingsChange = (values: any) => {
      const newValue = {
        ...(props.value || {}),
        ...(values || {}),
      }
      onChange?.(newValue);
    }

    return (
      <div className="flex gap-2 items-center">
        <DropdownList
          value={value}
          onChange={onSelectValueChange}
          placeholder="Please select"
          rowKey="id"
          data={formatData}
          renderItem={(item) => item.name}
          width={width || "100%"}
          dropdownWidth={width}
          renderLabel={(item) => {
            const provider = providers.find((p) => p.id === item.provider_id)
            return (
              <div className="flex items-center gap-2px">
                {
                  provider && (
                    <>
                      <span>{renderProvider(provider)}</span>
                      <span>/</span>
                    </>
                  )
                }
                <span>{item.name}</span>
              </div> 
             )
          }}
          searchKey="name"
          sorter={sorter}
          onSorterChange={setSorter}
          sorterOptions={[
            { label: "Name", key: "name" },
          ]}
          filters={filters}
          onFiltersChange={setFilters}
          filterOptions={filterOptions}
          defaultGroupVisible={true}
          groups={groups}
          onGroupsChange={(v)=>{setGroups(v)}}
          groupOptions={groupOptions}
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
        <div><ModelSettings onChange={onSettingsChange} value={value || {}} /></div>
      </div>
    )
}