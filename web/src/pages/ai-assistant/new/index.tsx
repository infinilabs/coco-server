
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
    // console.log(params);
  }


  return <div className="bg-white pt-15px pb-15px min-h-full">
      <div
        className="flex-col-stretch sm:flex-1-hidden">
        <div>
          <div className='mb-4 flex items-center text-lg font-bold'>
            <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
            <div>{t('route.ai-assistant_new')}</div>
          </div>
        </div>
        <div>
         <EditForm
          initialValues={initialValues}
          onSubmit={onSubmit}
          mode="new"
          loading={loading}
          />
        </div>
      </div>
  </div>
}