import { ChevronLeft, ChevronRight } from "lucide-react";
import clsx from "clsx";

interface PaginationProps {
  current: number;
  totalPage: number;
  onPrev: () => void;
  onNext: () => void;
  className?: string;
}

function Pagination({
  current,
  totalPage,
  onPrev,
  onNext,
  className = "",
}: PaginationProps) {
  return (
    <div
      className={clsx(
        "flex items-center justify-between h-8 px-2 text-[#666] dark:text-[#999] border-t border-input",
        className
      )}
    >
      <ChevronLeft
        className={clsx(
          "size-4 cursor-pointer hover:text-[#333] dark:hover:text-[#d8d8d8]",
          {
            "cursor-not-allowed opacity-50": current === 1,
          }
        )}
        onClick={onPrev}
      />

      <div className="text-xs">
        {current}/{totalPage}
      </div>

      <ChevronRight
        className={clsx(
          "size-4 cursor-pointer hover:text-[#333] dark:hover:text-[#d8d8d8]",
          {
            "cursor-not-allowed opacity-50": current === totalPage,
          }
        )}
        onClick={onNext}
      />
    </div>
  );
}

export default Pagination;
