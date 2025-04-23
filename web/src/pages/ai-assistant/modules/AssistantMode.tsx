
interface AssistantModeProps {
  value?: string;
  onChange?: (value: string) => void;
}

export const AssistantMode = (props: AssistantModeProps) =>{
  const {t} = useTranslation();
  const { value, onChange } = props;
  const modes = [{
    label: t('page.assistant.mode.simple'),
    value: 'simple'
  }, {
    label: t('page.assistant.mode.deep_think'),
    value: 'deep_think'
  },
  // {
  //   label: t('page.assistant.mode.workflow'),
  //   value: 'external_workflow'
  // }
];

  return (
    <div>
      <div className="flex gap-3">
        {modes.map((item) => (
          <div
            className={(item.value == value ? "border-[#1677FF] text-[#1677FF]": "border-gray-300") + " px-3 py-1 border rounded-4px cursor-pointer text-sm"}
            key={item.value}
            onClick={() => {
              if (onChange) {
                onChange(item.value);
              }
            }}
          >
            {item.label}
          </div>
        ))}
      </div>
    </div>
  );

}