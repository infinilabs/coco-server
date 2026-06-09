import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSize } from "ahooks";
import clsx from "clsx";
import { Attachments as AttachmentsList } from "@infinilabs/attachments";

import { useSpeechRecognition } from "./hooks/useSpeechRecognition";
import AutoResizeTextarea from "./AutoResizeTextarea";
import ChatIcons, { type SendMessageParams } from "./ChatIcons";
import InputControls from "./InputControls";
import { type DataSource } from "./SearchPopover";
import { useChatStore } from "./stores/chatStore";
import { Post, Upload } from "./api/axiosRequest";

interface ChatInputProps {
  onSend: (params: SendMessageParams) => void;
  onCancel: () => void;
  disabled: boolean;
  inputValue: string;
  changeInput: (val: string) => void;
  chatPlaceholder?: string;
  /** Max number of attachments allowed in a single message. Defaults to 5. */
  maxAttachments?: number;
  /** Endpoint used to upload attachments. Defaults to `/attachment/_upload`. */
  attachmentUploadUrl?: string;
  /** Accepted file types for the file picker (e.g. "image/*,.pdf"). */
  attachmentAccept?: string;
}

interface PendingAttachment {
  /** Local id used for list reconciliation; replaced by server id after upload. */
  localId: string;
  /** Server-assigned attachment id. Present after successful upload. */
  id?: string;
  name: string;
  size: number;
  status: "uploading" | "uploaded" | "error";
  error?: string;
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

type UploadResponse = {
  acknowledged?: boolean;
  result?: { attachments?: string[] };
  attachments?: string[];
};
import type { TFunction } from "i18next";

interface ChatInputProps {
  onSend: (params: SendMessageParams) => void;
  disabled: boolean;
  inputValue: string;
  changeInput: (val: string) => void;
  chatPlaceholder?: string;
  t?: TFunction;
  locale?: string;
}

const createGetDataSources = (ids?: string[]) => async (_query?: string): Promise<DataSource[]> => {
  let url = "/datasource/_search?from=0&size=10000&filter=enabled:any(true)";
  if (ids && ids.length > 0 && !ids.includes("*")) {
    url += `&filter=id:any(${ids.join(",")})`;
  }
  if (_query) {
    url += `&query=${encodeURIComponent(_query)}`;
  }
  const [err, res] = await Post<{ hits?: { hits?: Array<{ _id: string; _source?: Record<string, unknown> }> } }>(
    url,
    undefined
  );
  if (err || !res?.hits?.hits) return [];
  return res.hits.hits.map((hit) => ({
    id: hit._id,
    name: (hit._source?.name as string) ?? hit._id,
    ...(hit._source ?? {}),
  }));
};

const createGetMCPServers = (ids?: string[]) => async (_query?: string): Promise<DataSource[]> => {
  let url = "/mcp_server/_search?from=0&size=10000&filter=enabled:any(true)";
  if (ids && ids.length > 0 && !ids.includes("*")) {
    url += `&filter=id:any(${ids.join(",")})`;
  }
  if (_query) {
    url += `&query=${encodeURIComponent(_query)}`;
  }
  const [err, res] = await Post<{ hits?: { hits?: Array<{ _id: string; _source?: Record<string, unknown> }> } }>(
    url,
    undefined
  );
  if (err || !res?.hits?.hits) return [];
  return res.hits.hits.map((hit) => ({
    id: hit._id,
    name: (hit._source?.name as string) ?? hit._id,
    ...(hit._source ?? {}),
  }));
};

export default function ChatInput({
  onSend,
  onCancel,
  disabled = false,
  inputValue,
  changeInput,
  chatPlaceholder,
  maxAttachments = 5,
  attachmentUploadUrl = "/attachment/_upload",
  attachmentAccept,
  t: tProp,
  locale,
}: ChatInputProps) {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;

  const curChatEnd = useChatStore((state) => state.curChatEnd);
  const currentAssistant = useChatStore((state) => state.currentAssistant);

  // -------------------- Visibility by assistant type --------------------
  const assistantType = (currentAssistant?._source?.type as string) || "simple";
  // simple: search + mcp
  // deep_think: search + mcp + deepthink
  // deep_research: deep_research only
  const showSearch = assistantType === "simple" || assistantType === "deep_think";
  const showMCP = assistantType === "simple" || assistantType === "deep_think";
  const showDeepThink = assistantType === "deep_think";
  const showDeepResearch = assistantType === "deep_research";

  // -------------------- Datasource (from assistant config) --------------------
  const assistantDatasource = currentAssistant?._source?.datasource as
    | { enabled?: boolean; enabled_by_default?: boolean; ids?: string[]; visible?: boolean }
    | undefined;
  const [selectedDataSourceIds, setSelectedDataSourceIds] = useState<string[]>([]);
  const searchEnabled = (assistantDatasource?.enabled ?? true) && showSearch;
  const [isSearchActive, setIsSearchActive] = useState(
    () => !!(searchEnabled && assistantDatasource?.enabled_by_default)
  );
  useEffect(() => {
    setIsSearchActive(!!(searchEnabled && assistantDatasource?.enabled_by_default));
  }, [searchEnabled, assistantDatasource?.enabled_by_default]);
  const datasource = useMemo(() => ({
    enabled: searchEnabled,
    visible: (assistantDatasource?.visible ?? true) && showSearch,
  }), [searchEnabled, assistantDatasource?.visible, showSearch]);
  const getDataSources = useMemo(
    () => createGetDataSources(assistantDatasource?.ids),
    [assistantDatasource?.ids]
  );

  // -------------------- MCP (from assistant config) --------------------
  const assistantMcpServers = currentAssistant?._source?.mcp_servers as
    | { enabled?: boolean; enabled_by_default?: boolean; ids?: string[]; visible?: boolean }
    | undefined;
  const [selectedMCPIds, setSelectedMCPIds] = useState<string[]>([]);
  const mcpEnabled = (assistantMcpServers?.enabled ?? true) && showMCP;
  const [isMCPActive, setIsMCPActive] = useState(
    () => !!(mcpEnabled && assistantMcpServers?.enabled_by_default)
  );
  useEffect(() => {
    setIsMCPActive(!!(mcpEnabled && assistantMcpServers?.enabled_by_default));
  }, [mcpEnabled, assistantMcpServers?.enabled_by_default]);
  const mcp_servers = useMemo(() => ({
    enabled: mcpEnabled,
    visible: (assistantMcpServers?.visible ?? true) && showMCP,
  }), [mcpEnabled, assistantMcpServers?.visible, showMCP]);
  const getMCPServers = useMemo(
    () => createGetMCPServers(assistantMcpServers?.ids),
    [assistantMcpServers?.ids]
  );

  // -------------------- Deep Think (toggleable) --------------------
  const [isDeepThinkActive, setIsDeepThinkActive] = useState(true);
  const deepResearchActive = showDeepResearch;

  const textareaRef = useRef<{ reset: () => void; focus: () => void }>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const containerSize = useSize(containerRef);

  const [lineCount, setLineCount] = useState(1);
  const committedRef = useRef("");
  const {
    supported: speechSupported,
    listening,
    start,
    stop,
  } = useSpeechRecognition({
    lang: locale || "zh-CN",
    autoRestart: true,
    onInterim: (interim) => {
      const composed =
        committedRef.current +
        (interim ? (committedRef.current ? " " : "") + interim : "");
      changeInput(composed);
    },
    onFinal: (finalText) => {
      if (finalText) {
        committedRef.current =
          (committedRef.current ? committedRef.current + " " : "") + finalText;
        changeInput(committedRef.current);
      }
    },
  });

  const handleVoiceToggle = () => {
    if (listening) {
      stop();
      changeInput(committedRef.current);
    } else {
      committedRef.current = inputValue;
      start();
    }
  };

  // -------------------- Attachments --------------------
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [attachments, setAttachments] = useState<PendingAttachment[]>([]);

  const uploadingCount = attachments.filter((a) => a.status === "uploading").length;
  const uploadedIds = attachments
    .filter((a) => a.status === "uploaded" && a.id)
    .map((a) => a.id as string);
  const hasUploaded = uploadedIds.length > 0;

  const openFilePicker = useCallback(() => {
    if (disabled) return;
    if (attachments.length >= maxAttachments) return;
    fileInputRef.current?.click();
  }, [disabled, attachments.length, maxAttachments]);

  const removeAttachment = useCallback((id: string) => {
    setAttachments((list) => list.filter((a) => a.localId !== id && a.id !== id));
  }, []);

  const attachmentsData = useMemo(() => attachments.map((a) => ({
    id: a.id ?? a.localId,
    filename: a.name,
    extname: a.name.split(".").pop()?.toLowerCase(),
    size: formatBytes(a.size),
    status: a.status === "error" ? ("failed" as const) : (a.status as "uploading" | "analyzing" | "uploaded"),
    failedMessage: a.error,
  })), [attachments]);

  const uploadFile = useCallback(
    async (file: File, localId: string) => {
      const [err, res] = await Upload<UploadResponse>(attachmentUploadUrl, [file]);
      const serverIds =
        (res?.result?.attachments as string[] | undefined) ??
        (res?.attachments as string[] | undefined) ??
        [];
      setAttachments((list) =>
        list.map((a) => {
          if (a.localId !== localId) return a;
          if (err || serverIds.length === 0) {
            return {
              ...a,
              status: "error",
              error:
                (err as { message?: string } | undefined)?.message ||
                t("search.input.attachment_upload_failed") ||
                "Upload failed",
            };
          }
          return { ...a, status: "uploaded", id: serverIds[0] };
        })
      );
    },
    [attachmentUploadUrl, t]
  );

  const handleFilesPicked = useCallback(
    (files: FileList | null) => {
      if (!files || files.length === 0) return;
      const remaining = maxAttachments - attachments.length;
      const picked = Array.from(files).slice(0, Math.max(0, remaining));
      const additions: PendingAttachment[] = picked.map((f) => ({
        localId: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        name: f.name,
        size: f.size,
        status: "uploading",
      }));
      setAttachments((list) => [...list, ...additions]);
      picked.forEach((file, i) => {
        const localId = additions[i].localId;
        // Fire-and-forget; uploadFile updates state on completion.
        void uploadFile(file, localId);
      });
      // Reset the input so picking the same file again still fires onChange.
      if (fileInputRef.current) fileInputRef.current.value = "";
    },
    [attachments.length, maxAttachments, uploadFile]
  );
  // -----------------------------------------------------

  const appendFeatureParams = useCallback((params: SendMessageParams) => {
    if (!deepResearchActive) {
      params.deep_thinking = isDeepThinkActive;
      if (searchEnabled && isSearchActive) {
        params.search = true;
        params.datasource = selectedDataSourceIds;
      }
      if (mcpEnabled && isMCPActive) {
        params.mcp = true;
        params.mcp_servers = selectedMCPIds;
      }
    }
  }, [deepResearchActive, isDeepThinkActive, searchEnabled, isSearchActive, selectedDataSourceIds, mcpEnabled, isMCPActive, selectedMCPIds]);

  const handleSubmit = useCallback(() => {
    const trimmedValue = inputValue.trim();
    if (uploadingCount > 0) return;
    if (!trimmedValue && uploadedIds.length === 0) return;
    changeInput("");
    setAttachments([]);
    const newParams: SendMessageParams = {
      message: trimmedValue,
      attachments: uploadedIds.length > 0 ? uploadedIds : undefined,
    };
    appendFeatureParams(newParams);
    onSend(newParams);
  }, [inputValue, onSend, changeInput, uploadingCount, uploadedIds, appendFeatureParams]);

  const handleInputChange = useCallback(
    (value: string) => {
      changeInput(value);
      if (listening) {
        committedRef.current = value;
      }
    },
    [changeInput, listening]
  );

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const renderTextarea = () => {
    return (
      <AutoResizeTextarea
        ref={textareaRef}
        input={inputValue}
        setInput={handleInputChange}
        handleKeyDown={handleKeyDown}
        chatPlaceholder={chatPlaceholder}
        lineCount={lineCount}
        onLineCountChange={setLineCount}
        firstLineMaxWidth={containerSize?.width ?? 0}
        disabled={disabled}
      />
    );
  };

  const handleIconSend = (params: SendMessageParams) => {
    if (uploadingCount > 0) return;
    const trimmedValue = (params.message ?? "").trim();
    if (!trimmedValue && uploadedIds.length === 0) return;
    changeInput("");
    setAttachments([]);
    const newParams: SendMessageParams = {
      ...params,
      message: trimmedValue,
      attachments: uploadedIds.length > 0 ? uploadedIds : params.attachments,
    };
    appendFeatureParams(newParams);
    onSend(newParams);
  };

  const renderExtraIcon = () => (
    <div className="flex items-center gap-2 w-fit">
      <ChatIcons
        curChatEnd={curChatEnd}
        inputValue={inputValue}
        onSend={handleIconSend}
        onCancel={onCancel}
        speechSupported={speechSupported}
        listening={listening}
        onVoiceToggle={handleVoiceToggle}
        onAttachClick={
          attachments.length >= maxAttachments ? undefined : openFilePicker
        }
        attachmentCount={attachments.length}
        canSendWithoutText={hasUploaded}
        disableSend={uploadingCount > 0}
        t={t}
      />
    </div>
  );

  return (
    <div
      className={`w-full p-1 relative rounded-xl overflow-hidden border border-solid border-[#F0F0F0] dark:border-[#303030]`}
      style={{
        backgroundColor: 'var(--ant-color-bg-container)',
      }}
    >
      <input
        ref={fileInputRef}
        type="file"
        multiple
        accept={attachmentAccept}
        style={{ display: "none" }}
        onChange={(e) => handleFilesPicked(e.target.files)}
      />
      {attachments.length > 0 && (
        <div className="mb-2">
          <AttachmentsList
            data={attachmentsData}
            i18n={{
              labels: {
                uploading: t("search.input.uploading") || "上传中…",
                analyzing: t("search.input.analyzing") || "分析中…",
                failed:
                  t("search.input.attachment_upload_failed") || "上传失败",
              },
            }}
            onItemRemove={(item: { id: string }) => removeAttachment(item.id)}
          />
        </div>
      )}
      <div
        ref={containerRef}
        className={`rounded-sm flex items-center transition-all relative`}
        style={{
          backgroundColor: 'var(--ant-color-fill-quaternary)',
          color: 'var(--ant-color-text)',
        }}
      >
        <div
          className={clsx("min-h-[48px] w-full px-12px py-8px bg-transparent", {
            "flex items-center gap-2": lineCount === 1,
          })}
        >
          {renderTextarea()}

          {lineCount === 1 && renderExtraIcon()}

          {lineCount > 1 && (
            <div className="flex items-center mt-2">
              <div className="flex-1"></div>
              <div className="self-end">{renderExtraIcon()}</div>
            </div>
          )}
        </div>
      </div>

      {(showDeepThink || showDeepResearch || datasource.visible || mcp_servers.visible) && (
      <div className="pb-2">
        <InputControls
          isDeepThinkActive={isDeepThinkActive}
          setIsDeepThinkActive={setIsDeepThinkActive}
          isDeepResearchActive={deepResearchActive}
          showDeepThink={showDeepThink}
          showDeepResearch={showDeepResearch}
          datasource={datasource}
          selectedDataSourceIds={selectedDataSourceIds}
          onDataSourceSelectionChange={setSelectedDataSourceIds}
          isSearchActive={isSearchActive}
          setIsSearchActive={setIsSearchActive}
          getDataSources={getDataSources}
          mcp_servers={mcp_servers}
          selectedMCPIds={selectedMCPIds}
          onMCPSelectionChange={setSelectedMCPIds}
          isMCPActive={isMCPActive}
          setIsMCPActive={setIsMCPActive}
          getMCPByServer={getMCPServers}
          t={t}
        />
      </div>
      )}
    </div>
  );
}
