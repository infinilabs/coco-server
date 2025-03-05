import { Button } from "antd"

const ButtonRadio = (props) => {

    const { options = [], value, onChange } = props

    return (
        <div className="flex justify-between gap-24px">
            {
                options.map((item) => (
                    <Button variant="outlined" color={item.value === value ? 'primary' : 'default'} className="h-40px w-[calc((100%-24px)/2)]" onClick={() => onChange(item.value)}>
                        {item.label}
                    </Button>
                ))
            }
        </div>
    )
}

export default ButtonRadio