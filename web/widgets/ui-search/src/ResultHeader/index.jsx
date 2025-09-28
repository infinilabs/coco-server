import { ListFilter } from "lucide-react";

export function ResultHeader(props) {
  const { hits, isMobile } = props;
  return (
    <div className="flex justify-between items-center w-full text-[#999]">
      <div className="text-12px">
        Found {hits?.total || 0} records ({hits?.took || 0} millisecond)
      </div>
      {isMobile ? <ListFilter className="w-14px h-14px" /> : null}
    </div>
  );
}

export default ResultHeader;
