import { useState, useRef } from 'react';
import { Input, Button } from 'antd';
import { Send, Paperclip } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import clsx from 'clsx';

const { TextArea } = Input;

const ChatInput = (props) => {
  const { onSendMessage, disabled, placeholder } = props;
  const { t } = useTranslation();
  const [value, setValue] = useState('');
  const [isComposing, setIsComposing] = useState(false);
  const textareaRef = useRef(null);

  const handleSend = () => {
    if (value.trim() && !disabled) {
      onSendMessage(value.trim());
      setValue('');
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey && !isComposing) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="w-full flex flex-col gap-2 bg-white dark:bg-[#1f2937] rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm p-3">
      <TextArea
        ref={textareaRef}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        onCompositionStart={() => setIsComposing(true)}
        onCompositionEnd={() => setIsComposing(false)}
        placeholder={placeholder || t("chat.input.placeholder", "Ask whatever you want")}
        autoSize={{ minRows: 1, maxRows: 8 }}
        className="!border-0 !shadow-none !bg-transparent !px-0 !py-2 text-base resize-none focus:!shadow-none"
        disabled={disabled}
      />
      
      <div className="flex justify-between items-center pt-2 border-t border-gray-100 dark:border-gray-800">
        <div className="flex gap-2">
           <Button 
             type="text" 
             size="small" 
             className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
             icon={<Paperclip className="w-4 h-4" />}
           />
        </div>
        
        <Button
          type="primary"
          shape="circle"
          size="small"
          onClick={handleSend}
          disabled={!value.trim() || disabled}
          className={clsx("flex items-center justify-center w-8 h-8", {
            "bg-blue-600 hover:bg-blue-700": value.trim() && !disabled,
            "bg-gray-200 text-gray-400 border-none": !value.trim() || disabled
          })}
        >
          <Send className="w-4 h-4" />
        </Button>
      </div>
    </div>
  );
};

export default ChatInput;
