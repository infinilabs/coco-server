import { useEffect, useState } from "react";
import { Copy, Check } from "lucide-react";

interface CopyButtonProps {
  textToCopy: string;
}

export const CopyButton = ({ textToCopy }: CopyButtonProps) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      if (textToCopy && navigator.clipboard) {
        await navigator.clipboard.writeText(textToCopy);
        setCopied(true);
      }
    } catch {
    }
  };

  useEffect(() => {
    if (copied) {
      const t = setTimeout(() => setCopied(false), 1200);
      return () => clearTimeout(t);
    }
  }, [copied]);

  return (
    <button
      className={`bg-transparent border-0 cursor-pointer p-4px hover:bg-black/5 dark:hover:bg-white/5 rounded-lg transition-colors`}
      onClick={handleCopy}
    >
      {copied ? (
        <Check className="w-4 h-4 text-[#38C200] dark:text-[#38C200]" />
      ) : (
        <Copy className="w-4 h-4 text-gray-600 dark:text-gray-300" />
      )}
    </button>
  );
};
