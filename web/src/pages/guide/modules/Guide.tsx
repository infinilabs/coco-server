import { Button, Flex, Form } from "antd";
import UserForm from "./UserForm";
import LLMForm from "./LLMForm";
import { useLoading } from '@sa/hooks';
import { setup } from "@/service/api/guide";

const Guide = memo(() => {
    const [form] = Form.useForm();
    const [step, setStep] = useState(0);
    const router = useRouterPush();
    const { endLoading, loading, startLoading } = useLoading();
    const [editValue, setEditValue] = useState()

    const handleSubmit = async (isPass?: boolean) => {
      if (step === 0) {
        const params = await form.validateFields();
        setStep(1)
        setEditValue(params)
      } else if (step === 1) {
        let body
        if (isPass) {
          body = editValue
        } else {
          const params = await form.validateFields();
          body = {
            ...(editValue || {}),
            ...params
          }
        }
        startLoading()
        const { error } = await setup(body);
        endLoading()
        if (!error) {
          router.routerPushByKey('login');
        }
      }
    }

    const renderContent = () => {
      switch (step) {
        case 0:
          return <UserForm form={form} onSubmit={handleSubmit}/>
        case 1:
          return <LLMForm form={form} onSubmit={handleSubmit} loading={loading}/>
        default:
          break;
      }
    }

    return (
      <div className="size-full flex flex-col items-left justify-center px-10%">
        <Flex gap="8px" wrap className="m-b-32px">
            <Button onClick={() => setStep(0)} type={step === 0 ? "primary" : "default"} size="small" className="h-24px rounded-8px">1</Button>
            <Button onClick={() => handleSubmit()} type={step === 1 ? "primary" : "default"} size="small" className="h-24px rounded-8px">2</Button>
        </Flex>
        <div className="w-440px">
          {renderContent()}
        </div>
      </div>
    )
})

export default Guide;