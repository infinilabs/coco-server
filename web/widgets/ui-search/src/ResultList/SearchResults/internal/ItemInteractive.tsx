import type React from "react";

import { joinRel } from "./joinRel";

export function ItemInteractive({
  href,
  target,
  rel,
  onClick,
  className,
  children
}: {
  href?: string;
  target?: string;
  rel?: string;
  onClick?: () => void;
  className?: string;
  children: React.ReactNode;
}) {
  const interactiveClassName = className ? `${className} cursor-pointer` : "cursor-pointer";

  if (href) {
    return (
      <a
        href={href}
        target={target}
        rel={joinRel(rel, target)}
        className={interactiveClassName}
        onClick={() => onClick?.()}
      >
        {children}
      </a>
    );
  }

  return (
    <button type="button" className={interactiveClassName} onClick={onClick}>
      {children}
    </button>
  );
}
