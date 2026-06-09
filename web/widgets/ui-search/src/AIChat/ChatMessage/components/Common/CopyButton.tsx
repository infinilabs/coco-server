import { useState } from "react";
import { Copy, Check } from "lucide-react";

import { copyToClipboard } from "../../utils";

interface CopyButtonProps {
  textToCopy: string;
}

export const CopyButton = ({ textToCopy }: CopyButtonProps) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await copyToClipboard(textToCopy);
      setCopied(true);
      const timerID = setTimeout(() => {
        setCopied(false);
        clearTimeout(timerID);
      }, 2000);
    } catch (err) {
      console.error("copy error:", err);
    }
  };

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
