
import {EditForm} from "../modules/EditForm";
import {createMCPServer} from "@/service/api/mcp-server";

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const initialValues = {
    enabled: true,
    name: '',
    type: "sse",
  };
  const [loading, setLoading] = useState(false);

  const onSubmit = (values: any) => {
    setLoading(true);
    createMCPServer(values).then((res)=>{
      if(res?.data?.result === 'created') {
        window.$message?.success(t('common.addSuccess'));
        nav(`/mcp-server/list`)
      }
    }).finally(()=>{
      setLoading(false);
    })
    // console.log(values);
  }

  return (
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        className="min-h-full flex-col-stretch sm:flex-1-hidden card-wrapper"
      >
        <div className="mb-4 ml--16px flex items-center text-lg font-bold">
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