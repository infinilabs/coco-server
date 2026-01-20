import ChatHeader from "../ChatHeader";
import ChatLayout from "../Layout/ChatLayout";
import { History, Chat as AIChat, AssistantList, ChatInput } from "@infinilabs/ai-chat";

export default function Chat({
    commonProps,
    onNewChat,
    language,
    apiConfig,
    onBackToSearch,
    queryParams
}) {
    const { BaseUrl, Token, endpoint } = apiConfig || {};

    const chatRef = useRef(null);

    const [isHistoryOpen, setIsHistoryOpen] = useState(true);
    const [inputValue, setInputValue] = useState("");
    const [isDeepThinkActive, setIsDeepThinkActive] = useState(false);

    const onSendMessage = async (params) => {
        if (chatRef.current) {
            chatRef.current.init(params);
        }
    };

    const handleNewChat = () => {
        if (onNewChat) {
            onNewChat();
        } else if (chatRef.current) {
            chatRef.current.clearChat();
        }
    };

    return (
        <ChatLayout
            {...commonProps}
            content={
                <AIChat
                    ref={chatRef}
                    BaseUrl={BaseUrl}
                    formatUrl={(data) => `${endpoint}${BaseUrl}${data.url}`}
                    Token={Token}
                    locale={language === 'zh-CN' ? 'zh' : 'en'}
                />
            }
            input={
                <ChatInput
                    onSend={onSendMessage}
                    disabled={false}
                    isChatMode={true}
                    inputValue={inputValue}
                    changeInput={setInputValue}
                    isDeepThinkActive={isDeepThinkActive}
                    setIsDeepThinkActive={setIsDeepThinkActive}
                    chatPlaceholder={language === 'zh-CN' ? '请输入问题...' : 'Type a message...'}
                />
            }
            sidebarCollapsed={!isHistoryOpen}
            header={
                <ChatHeader
                    onNewChat={handleNewChat}
                    showChatHistory={true}
                    onToggleHistory={() => setIsHistoryOpen(open => !open)}
                    AssistantList={
                        <AssistantList
                            BaseUrl={BaseUrl}
                            Token={Token}
                            locale={language === 'zh-CN' ? 'zh' : 'en'}
                        />
                    }
                    onBackToSearch={onBackToSearch}
                />
            }
            sidebar={
                <History
                    BaseUrl={BaseUrl}
                    Token={Token}
                    locale={language === 'zh-CN' ? 'zh' : 'en'}
                />
            }
        />
    );
}