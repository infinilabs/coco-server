import { type ReactNode } from "react";
import clsx from "clsx";

interface CommonLabelProps {
  isActive: boolean;
  setIsActive?: (val: boolean) => void;
  icon: ReactNode;
  label: string;
  title?: string;
}

export default function CommonLabel({
  isActive,
  setIsActive,
  icon,
  label,
  title,
}: CommonLabelProps) {
  return (
    <div
      className={clsx(
        "flex items-center justify-center gap-1 h-6 px-2 rounded-full cursor-pointer",
        !isActive && setIsActive && "hover:bg-[#EDEDED] dark:hover:bg-[#202126]"
      )}
      style={{
        backgroundColor: isActive
          ? 'var(--ant-color-primary-bg)'
          : 'transparent',
        transition: 'background-color 0.3s ease',
      }}
      onClick={() => setIsActive?.(!isActive)}
      title={title || label}
    >
      {icon}

      <div
        className="overflow-hidden"
        style={{
          maxWidth: isActive ? '200px' : '0px',
          opacity: isActive ? 1 : 0,
          transition: 'max-width 0.3s ease, opacity 0.2s ease',
        }}
      >
        <span className="text-xs whitespace-nowrap" style={{ color: 'var(--ant-color-primary)' }}>
          {label}
        </span>
      </div>
    </div>
  );
}
