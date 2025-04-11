
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


  return <div className="bg-white pt-15px pb-15px min-h-full">
      <div
        className="flex-col-stretch sm:flex-1-hidden">
        <div>
          <div className='mb-4 flex items-center text-lg font-bold'>
            <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
            <div>{t('route.mcp-server_new')}</div>
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