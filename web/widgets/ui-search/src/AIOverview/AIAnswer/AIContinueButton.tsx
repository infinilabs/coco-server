import { MessageCircle } from "lucide-react";

export function AIContinueButton({
  label = "继续追问",
  onClick,
  disabled = false,
}: {
  label?: string;
  onClick?: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      className="bg-transparent border-0 inline-flex items-center gap-1 text-sm font-medium text-[#1784FC] cursor-pointer hover:brightness-110 focus:outline-none focus-visible:ring-2 focus-visible:ring-[#1677ff]/40 dark:focus-visible:ring-slate-600 disabled:opacity-40 disabled:cursor-not-allowed disabled:pointer-events-none"
      disabled={disabled}
      onClick={onClick}
    >
      <MessageCircle className="h-4 w-4" />
      {label}
    </button>
  );
}
