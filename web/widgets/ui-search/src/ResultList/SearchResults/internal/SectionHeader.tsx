import clsx from "clsx";

import type React from "react";

// function getSourceBadgeText(source: string) {
//   const normalized = source.trim().toLowerCase();
//   if (!normalized) return "";
//   if (normalized === "google") return "G";
//   return source.trim().slice(0, 1).toUpperCase();
// }

export function SectionHeader({
  title,
  titleIcon,
  titleIconBgColor,
  source,
  className,
  titleClassName,
  titleAction,
}: {
  title?: string;
  titleIcon?: React.ReactNode;
  titleIconBgColor?: string;
  source?: string;
  className?: string;
  titleClassName?: string;
  titleAction?: {
    label: string;
    href?: string;
    target?: string;
    rel?: string;
    onClick?: () => void;
  };
}) {
  if (!title && !titleIcon && !source) return null;

  return (
    <div className={clsx("mb-2 flex min-w-0 items-center gap-2", className)}>
      {titleIcon ? (
        <span
          className={clsx(
            "inline-flex h-5 w-5 flex-none items-center justify-center",
            titleIconBgColor && "rounded-md text-white"
          )}
          style={titleIconBgColor ? { backgroundColor: titleIconBgColor } : undefined}
        >
          {titleIcon}
        </span>
      ) : null}

      {title ? (
        titleAction?.href ? (
          <a
            className={clsx(
              "min-w-0 cursor-pointer text-xl",
              titleClassName
            )}
            href={titleAction.href}
            target={titleAction.target}
            rel={titleAction.rel}
            onClick={() => titleAction.onClick?.()}
          >
            {title}
          </a>
        ) : titleAction?.onClick ? (
          <button
            type="button"
            className={clsx(
              "min-w-0 cursor-pointer border-0 bg-transparent p-0 text-left text-xl",
              titleClassName
            )}
            onClick={titleAction.onClick}
          >
            {title}
          </button>
        ) : (
          <span
            className={clsx(
              "min-w-0 text-xl",
              titleClassName
            )}
          >
            {title}
          </span>
        )
      ) : null}

      {/* {source ? (
        <span
          className="inline-flex h-6 w-6 flex-none items-center justify-center rounded-md bg-slate-100 text-xs font-semibold text-slate-700 dark:bg-slate-800 dark:text-slate-200"
          title={source}
        >
          {getSourceBadgeText(source)}
        </span>
      ) : null} */}
    </div>
  );
}
