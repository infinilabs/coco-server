import { Button, Flex, Form } from "antd";
import UserForm from "./UserForm";
import LLMForm from "./LLMForm";
import { useRouter } from '@sa/simple-router';

const Guide = memo(() => {
    const router = useRouter();
    const [form] = Form.useForm();
    const [step, setStep] = useState(0);

    const renderContent = () => {
      switch (step) {
        case 0:
          return <UserForm form={form} onSubmit={() => setStep(1)}/>
        case 1:
          return <LLMForm form={form} onSubmit={() => router.push('/login')}/>
        default:
          break;
      }
    }

    return (
      <div className="size-full flex flex-col items-left justify-center px-10%">
        <Flex gap="8px" wrap className="m-b-32px">
            <Button type={step === 0 ? "primary" : "default"} size="small" className="h-24px rounded-8px">1</Button>
            <Button type={step === 1 ? "primary" : "default"} size="small" className="h-24px rounded-8px">2</Button>
        </Flex>
        <div className="w-440px">
          {renderContent()}
        </div>
      </div>
    )
})

export default Guide;