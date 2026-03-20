import { ListFilter, PanelRightOpen } from "lucide-react";

export function ResultHeader(props) {
  const { hits, isMobile, siderCollapse, setSiderCollapse } = props;
  return (
    <div className="flex gap-8px items-center w-full text-[#999]">
      <PanelRightOpen className="w-16px h-16px cursor-pointer" onClick={() => setSiderCollapse(!siderCollapse)} />
      <div className="text-12px">
        Found {hits?.total || 0} records ({hits?.took || 0} millisecond)
      </div>
      {isMobile ? <ListFilter className="w-14px h-14px" /> : null}
    </div>
  );
}

export default ResultHeader;
