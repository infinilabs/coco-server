import type React from "react";

import { joinRel } from "./joinRel";

export function TitleInteractive({
  href,
  target,
  rel,
  onClick,
  children
}: {
  href?: string;
  target?: string;
  rel?: string;
  onClick?: () => void;
  children: React.ReactNode;
}) {
  if (href) {
    return (
      <a
        href={href}
        target={target}
        rel={joinRel(rel, target)}
        onClick={() => onClick?.()}
        className="group flex w-full min-w-0 items-center gap-2 cursor-pointer"
      >
        {children}
      </a>
    );
  }

  if (onClick) {
    return (
      <button
        type="button"
        onClick={onClick}
        className="group flex w-full min-w-0 items-center gap-2 text-left cursor-pointer"
      >
        {children}
      </button>
    );
  }

  return <div className="flex w-full min-w-0 items-center gap-2">{children}</div>;
}
