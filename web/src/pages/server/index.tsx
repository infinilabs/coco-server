import { selectUserInfo } from "@/store/slice/auth";
import { Button, Card } from "antd";

const SETTINGS = [
  {
    key: 'llm',
    title: 'LLMs',
    desc: 'Connect the large model to enable AI chat, intelligent search, and a work assistant.',
    icon: <SvgIcon icon="mdi:settings-outline"/>,
    link: '/settings?tab=llm'
  },
  {
    key: 'data-source',
    title: 'Data source',
    desc: 'Add data sources to the service list for unified search and analysis.',
    icon: <SvgIcon icon="mdi:plus-thick"/>,
    link: ''
  },
  {
    key: 'ai-assistant',
    title: 'AI Assistant',
    desc: 'Set a personalized AI assistant to handle tasks efficiently and provide intelligent suggestions.',
    icon: <SvgIcon icon="mdi:plus-thick"/>,
    link: ''
  }
]

export function Component() {

    const userInfo = useAppSelector(selectUserInfo);
    const routerPush = useRouterPush();

    return (
      <div>
        <Card className="m-b-12px px-32px py-40px" classNames={{ body: "!p-0" }}>
          <div className="flex items-center m-b-12px">
            <div className="m-r-16px text-32px color-#333">
              {userInfo.userName}'s Coco Server
            </div>
            <Button type="link" className="w-40px h-40px bg-#F7F9FC !hover:bg-#F7F9FC rounded-12px p-0"><SvgIcon className="text-24px" icon="mdi:square-edit-outline"/></Button>
          </div>
          <div className="m-b-48px color-#888">
            Coco AI Server - Search, Connect, Collaborate, AI-powered enterprise search, all in one space.
          </div>
          <div className="m-b-16px text-20px color-#333">
            Server address
          </div>
          <div className="flex m-b-16px">
            <div className="w-400px h-48px color-#333 m-r-8px bg-#F7F9FC px-12px leading-48px rounded-4px overflow-auto"> 
              https://coco.infini.cloud/
            </div>
            <Button className="w-100px h-48px" type="primary"><SvgIcon className="text-24px" icon="mdi:content-copy" /></Button>
          </div>
          <div className="m-b-16px color-#888">
            In the connect settings of Coco AI, adding the Server address to the service list will allow you to access the service in Coco AI
          </div>
          <Button type="link" className="px-0">Download Coco  AI <SvgIcon className="text-16px" icon="mdi:external-link"/></Button>
        </Card>
        <Card className="p-32px" classNames={{ body: "flex gap-32px justify-start !p-0 -mx-32px" }}>
          {
            SETTINGS.map((item) => (
              <div key={item.key} className="basis-1/3">
                <div className="text-20px color-#333 m-b-16px">{item.title}</div>
                <div className="color-#888 m-b-45px">{item.desc}</div>
                <Button onClick={() => item.link && routerPush.routerPush(item.link)} type="primary" className="w-40px h-40px rounded-12px text-24px p-0">{item.icon}</Button>
              </div>
            ))
          }
        </Card>
      </div>
    );
}
