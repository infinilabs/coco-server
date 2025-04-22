
import {EditForm} from "../modules/EditForm";
import {createAssistant} from "@/service/api/assistant";

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const initialValues = {
    enabled: true,
    chat_settings: {
      greeting_message: "Hi! I’m Coco, nice to meet you. I can help answer your questions by tapping into the internet and your data sources. How can I assist you today?",
      suggested: {
        enabled: false,
        questions: [],
      },
      history_message:{
        compression_threshold: 1000,
        summary: true,
        number: 5,
      },
    },
    model_settings: {
      temperature: 0.5,
      top_p: 0.5,
      presence_penalty: 0,
      frequency_penalty: 0,
      max_tokens: 4000,
    },
    datasource:{
      ids: ["*"],
      enabled: true,
      visible: true
    },
    mcp_servers:{
      ids: ["*"],
      enabled: true,
      visible: true
    },
    keepalive: "30m",
    type: "simple",
  };
  const [loading, setLoading] = useState(false);

  const onSubmit = (values: any) => {
    const params = {
      ...values,
      datasource: {
        ...(values.datasource || {}),
        ids: values.datasource?.ids?.includes('*') ? ['*'] : values.datasource?.ids,
      }
    };
   
    setLoading(true);
    createAssistant(params).then((res)=>{
      if(res?.data?.result === 'created') {
        window.$message?.success(t('common.addSuccess'));
        nav(`/ai-assistant/list`)
      }
    }).finally(()=>{
      setLoading(false);
    })
  }

  return (
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        className="min-h-full flex-col-stretch sm:flex-1-hidden card-wrapper"
      >
        <div className="mb-30px ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          <div>{t(`route.ai-assistant_edit`)}</div>
        </div>
        <div className="px-30px">
          <EditForm
            initialValues={initialValues}
            onSubmit={onSubmit}
            mode="new"
            loading={loading}
          />
        </div>
      </ACard>
    </div>
  )
}