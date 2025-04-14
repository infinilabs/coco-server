import { Button } from 'antd';

const ButtonRadio = props => {
  const { onChange, options = [], value } = props;

  return (
    <div className="flex justify-between gap-24px">
      {options.map(item => (
        <Button
          className="h-40px w-[calc((100%-24px)/2)]"
          color={item.value === value ? 'primary' : 'default'}
          key={item.value}
          variant="outlined"
          onClick={() => onChange(item.value)}
        >
          {item.label}
        </Button>
      ))}
    </div>
  );
};

export default ButtonRadio;
