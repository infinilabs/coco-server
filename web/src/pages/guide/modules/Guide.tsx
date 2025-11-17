import { useLoading } from '@sa/hooks';
import { Button, Flex, Form } from 'antd';

import { setup } from '@/service/api/guide';

import LLMForm from './LLMForm';
import UserForm from './UserForm';

const Guide = memo(() => {
  const [form] = Form.useForm();
  const [step, setStep] = useState(0);
  const router = useRouterPush();
  const { endLoading, loading, startLoading } = useLoading();
  const [editValue, setEditValue] = useState();

  const handleSubmit = async (isPass?: boolean) => {
    if (step === 0) {
      const params = await form.validateFields();
      const { confirm_password, ...rest } = params
      setStep(1);
      setEditValue(rest);
    } else if (step === 1) {
      let body;
      if (isPass) {
        body = editValue;
      } else {
        const params = await form.validateFields();
        const { confirm_password, ...rest } = params
        if(typeof params.llm.reasoning === "undefined") {
          params.llm.reasoning = params.llm.type === "deepseek";
        }
        body = {
          ...(editValue || {}),
          ...rest
        };
      }
      startLoading();
      const { error } = await setup(body);
      endLoading();
      if (!error) {
        router.routerPushByKey('login');
      }
    }
  };

  const renderContent = () => {
    switch (step) {
      case 0:
        return (
          <UserForm
            form={form}
            onSubmit={handleSubmit}
          />
        );
      case 1:
        return (
          <LLMForm
            form={form}
            loading={loading}
            onSubmit={handleSubmit}
          />
        );
      default:
        break;
    }
  };

  return (
    <div className="items-left size-full flex flex-col justify-center overflow-auto px-10%">
      <Flex
        wrap
        className="m-b-32px"
        gap="8px"
      >
        <Button
          className="h-24px rounded-8px"
          size="small"
          type={step === 0 ? 'primary' : 'default'}
          onClick={() => setStep(0)}
        >
          1
        </Button>
        <Button
          className="h-24px rounded-8px"
          size="small"
          type={step === 1 ? 'primary' : 'default'}
          onClick={() => handleSubmit()}
        >
          2
        </Button>
      </Flex>
      <div className="w-440px">{renderContent()}</div>
    </div>
  );
});

export default Guide;
