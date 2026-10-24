import { Tooltip } from "antd";
import dayjs from "dayjs";
import { useTranslation } from "react-i18next";

import "dayjs/locale/zh-cn";

import { formatDate } from "../../../utils/date";
import { MetaDot } from "./MetaDot";

export function AuthorDate({ author, date }: { author?: string; date?: string }) {
  const { i18n } = useTranslation();
  if (!author && !date) return null;

  // `date` is a raw timestamp — show a short relative time ("10 天前") and keep
  // the exact time on hover; values that don't parse fall back to raw display
  const parsed = date ? dayjs(date) : undefined;
  const relative = parsed?.isValid()
    ? parsed
        .locale(i18n.language?.toLowerCase().startsWith("zh") ? "zh-cn" : "en")
        .fromNow()
    : date;
  const full = parsed?.isValid() ? formatDate(date) : undefined;

  return (
    <div className="flex min-w-0 items-center gap-x-6px text-xs">
      {author ? <span className="min-w-0 truncate">{author}</span> : null}
      {author && relative ? <MetaDot /> : null}
      {relative ? (
        full ? (
          <Tooltip title={full}>
            <span className="cursor-default whitespace-nowrap">{relative}</span>
          </Tooltip>
        ) : (
          <span className="whitespace-nowrap">{relative}</span>
        )
      ) : null}
    </div>
  );
}
