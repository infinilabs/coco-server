import { Button } from "antd";
import Clipboard from "clipboard";

import { localStg } from "@/utils/storage";

const CocoAI = ({
  provider,
  requestID,
}: {
  provider: string | null;
  requestID: string | null;
}) => {
  const { t } = useTranslation();

  const token = localStg.get("token");
  const url = token
    ? `coco://oauth_callback?code=${token}&request_id=${requestID}&provider=${provider}`
    : "";
  const linkRef = useRef<HTMLButtonElement>(null);
  const copyRef = useRef<HTMLButtonElement>(null);
  const router = useRouterPush();

  const initClipboard = (text?: string) => {
    if (!copyRef.current || !text) return;

    const clipboard = new Clipboard(copyRef.current, {
      text: () => text,
    });

    clipboard.on("success", () => {
      window.$message?.success(t("common.copySuccess"));
    });
  };

  useEffect(() => {
    setTimeout(() => {
      linkRef.current?.click();
    }, 5000);
  }, []);

  useEffect(() => {
    if (copyRef.current) {
      initClipboard(url);
    }
  }, [url, copyRef.current]);

  return (
    <>
      <div style={{ wordBreak: "break-all" }} className="text-16px m-b-12px">
        {t("page.login.cocoAI.autoDesc")}
      </div>
      <Button ref={linkRef} type="link" className="m-b-16px p-0" href={url}>
        {t("page.login.cocoAI.launchCocoAI")}
      </Button>
      <div style={{ wordBreak: "break-all" }} className="text-16px m-b-12px">
        {t("page.login.cocoAI.copyDesc")}
      </div>
      <div className="m-b-16px relative group">
        <pre
          style={{ wordBreak: "break-all" }}
          className="relative bg-gray-100 border border-gray-300 rounded-4px p-8px whitespace-pre-wrap"
        >
          <code className="text-gray-700">{url}</code>
        </pre>
        <Button
          ref={copyRef}
          type="link"
          className="z-1 absolute right-0 top-0 p-4px group-hover:block hidden"
        >
          <SvgIcon className="text-16px" icon="mdi:content-copy" />
        </Button>
      </div>
      <div style={{ wordBreak: "break-all" }} className="text-16px m-b-12px">
        {t("page.login.cocoAI.enterCocoServerDesc")}
      </div>
      <Button
        onClick={() => router.routerPushByKey("home")}
        type="link"
        className="p-0"
      >
        {t("page.login.cocoAI.enterCocoServer")}
      </Button>
    </>
  );
};

export default CocoAI;
