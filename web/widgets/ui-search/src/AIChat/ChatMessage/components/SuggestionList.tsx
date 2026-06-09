import { MoveRight } from "lucide-react";

interface SuggestionListProps {
  suggestions: string[];
  onSelect: (suggestion: string) => void;
}

export function SuggestionList({ suggestions, onSelect }: SuggestionListProps) {
  if (!suggestions || suggestions.length === 0) return null;

  return (
    <div className="mt-24px flex flex-col gap-16px">
      {suggestions.map((suggestion, index) => (
        <button
          key={index}
          onClick={() => onSelect(suggestion)}
          className="text-left inline-flex items-center px-3 py-1.5 rounded-12px bg-transparent border border-solid border-[#F0F0F0] dark:border-[#303030] text-[#666] dark:text-white/80 hover:bg-[#EDEDED] dark:hover:bg-[#3A3A3A] transition-colors w-fit max-w-full wrap-break-word whitespace-pre-wrap"
        >
          <span className="break-all">{suggestion}</span>
          <MoveRight className="w-3 h-3 ml-1.5 shrink-0" />
        </button>
      ))}
    </div>
  );
}
