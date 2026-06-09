import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import clsx from "clsx";
import { debounce } from "lodash";
import { I18nextProvider, useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import { ChevronDown, Search, Check } from "lucide-react";

import i18n from "../i18n";
import { Button, Input, type InputRef } from "antd";
import { Post } from "./api/axiosRequest";
import { useChatStore } from "./stores/chatStore";
import FontIcon from "./FontIcon";
import { useIconfontScript } from "./hooks/useScript";
import RefreshIcon from "../icons/RefreshIcon";

interface AssistantListProps {
  BaseUrl: string;
  Token?: string;
  assistantIDs?: string[];
  locale?: string;
  t?: TFunction;
  headers?: Record<string, string>;
}

interface AssistantHit {
  _id: string;
  _source?: {
    name?: string;
    icon?: string;
    [key: string]: unknown;
  };
}

function InnerAssistantList({ assistantIDs = [], locale = "en", t: tProp }: AssistantListProps) {
  useIconfontScript();
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;
  const currentAssistant = useChatStore((state) => state.currentAssistant);
  const setCurrentAssistant = useChatStore((state) => state.setCurrentAssistant);
  const setAssistantList = useChatStore((state) => state.setAssistantList);

  const [assistants, setAssistants] = useState<AssistantHit[]>([]);

  useEffect(() => {
    setAssistantList(assistants);
  }, [assistants, setAssistantList]);

  // 当 currentAssistant 只有 _id 没有 _source 时，从本地列表补全信息
  useEffect(() => {
    if (currentAssistant?._id && !currentAssistant._source && assistants.length > 0) {
      const match = assistants.find((a) => a._id === currentAssistant._id);
      if (match) {
        setCurrentAssistant({ _id: match._id, _source: match._source });
      }
    }
  }, [currentAssistant, assistants, setCurrentAssistant]);

  const [isRefreshing, setIsRefreshing] = useState(false);
  const [open, setOpen] = useState(false);
  const searchInputRef = useRef<InputRef>(null);
  const [keyword, setKeyword] = useState("");
  const [inputValue, setInputValue] = useState("");
  const [total, setTotal] = useState(0);
  
  const debouncedKeyword = useMemo(
    () => debounce((k: string) => {
      setKeyword(k);
      setAssistants([]);
      setTotal(0);
    }, 500),
    []
  );

  const assistantIDsStr = JSON.stringify(assistantIDs);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const stableAssistantIDs = useMemo(() => assistantIDs, [assistantIDsStr]);

  const fetchAssistant = useCallback(
    async () => {
      try {
        const queryParams = [`from=0`, `size=10000`];
        if (keyword) queryParams.push(`query=${encodeURIComponent(keyword)}`);
        if (stableAssistantIDs.length) queryParams.push(`filter=id:any(${encodeURIComponent(stableAssistantIDs.join(","))})`);

        const [error, res] = await Post<{
          hits?: { hits?: AssistantHit[], total?: { value: number } };
        }>(`/assistant/_search?${queryParams.join("&")}`, undefined);

        if (error) {
          console.error("_search", error);
          return;
        }

        const list = (res?.hits?.hits ?? []) as AssistantHit[];
        const totalValue = res?.hits?.total?.value ?? 0;

        setAssistants(list);
        setTotal(totalValue);

        if (list.length > 0) {
          const current = useChatStore.getState().currentAssistant;
          // If no assistant is selected, select the first one
          if (!current?._id) {
            setCurrentAssistant({
              _id: list[0]._id,
              _source: list[0]._source,
            });
          }
        }
      } catch (e) {
        console.error(e);
      }
    },
    [keyword, stableAssistantIDs, setCurrentAssistant]
  );

  useEffect(() => {
    const t = setTimeout(() => {
      fetchAssistant();
    }, 0);
    return () => clearTimeout(t);
  }, [fetchAssistant]);

  const handleRefresh = async () => {
    setIsRefreshing(true);
    await fetchAssistant();
    setTimeout(() => setIsRefreshing(false), 800);
  };



  return (
    <div className="relative">
      <Button
        shape="round"
        className="!rounded-12px border-[#F0F0F0] dark:border-[#303030] !px-8px"
        onClick={() => {
          setOpen((v) => !v);
          setTimeout(() => {
            if (searchInputRef.current) searchInputRef.current.focus();
          }, 0);
        }}
      >
        {currentAssistant?._source?.icon ? (
          currentAssistant._source.icon.startsWith("font_") ? (
            <FontIcon
              name={currentAssistant._source.icon}
              className="w-4 h-4 mr-1"
            />
          ) : (
            <img
              src={currentAssistant._source.icon}
              className="w-4 h-4 mr-1"
              alt="assistant"
              onError={(e) => {
                const el = e.currentTarget as HTMLImageElement;
                el.style.display = "none";
              }}
            />
          )
        ) : null}
        <span className="text-sm">
          {currentAssistant?._source?.name || t("assistant_list.default_name")}
        </span>
        <ChevronDown className="h-4 w-4" />
      </Button>

      {open && (
        <div
          className="border-[#F0F0F0] dark:border-[#303030] border border-solid absolute left-0 top-full z-50 mt-2 w-64 rounded-xl shadow-lg p-3"
          style={{
            backgroundColor: 'var(--ant-color-bg-elevated)',
            color: 'var(--ant-color-text)',
          }}
          onMouseMove={() => {
            // no-op
          }}
        >
          <div className="flex items-center justify-between text-sm font-semibold mb-2" style={{ color: 'var(--ant-color-text)' }}>
            <div className="truncate">
              {t("assistant_list.title")}（{total}）
            </div>
            <button
              className="bg-transparent border-0 h-6 w-6 flex items-center justify-center cursor-pointer rounded transition-colors"
              style={{ color: 'var(--ant-color-text)' }}
              type="button"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                handleRefresh();
              }}
              disabled={isRefreshing}
            >
              <RefreshIcon
                className={clsx("w-4 h-4", {
                  "animate-spin": isRefreshing,
                })}
                style={{ color: 'var(--ant-color-text-secondary)' }}
              />
            </button>
          </div>

          <div className="mt-2">
            <Input
              ref={searchInputRef}
              autoFocus
              value={inputValue}
              placeholder={t("assistant_list.search.placeholder")}
              className="h-8 border-[#F0F0F0] dark:border-[#303030] rounded-12px"
              style={{ backgroundColor: 'var(--ant-color-fill-quaternary)' }}
              prefix={<Search className="text-[#999] h-4 w-4" />}
              onChange={(event) => {
                const val = event.target.value;
                setInputValue(val);
                debouncedKeyword(val);
              }}
              onClick={(e) => e.stopPropagation()}
            />
          </div>

          <div
            className="mt-2 max-h-60 overflow-auto custom-scrollbar"
          >
            {assistants.length > 0 ? (
              <div className="flex flex-col gap-1">
                {assistants.map((assistant) => {
                  const name = assistant._source?.name || assistant._id;
                  const isActive = currentAssistant?._id === assistant._id;
                  return (
                    <button
                      key={assistant._id}
                      className={clsx(
                        "cursor-pointer bg-transparent border-0 hover:bg-[#EDEDED] dark:hover:bg-[#353F4D] w-full text-left flex items-center gap-2 px-2 py-1.5 rounded-lg text-sm transition-colors",
                        isActive ? "font-medium" : ""
                      )}
                      style={{
                        backgroundColor: isActive ? 'var(--ant-color-primary-bg)' : undefined,
                        color: isActive ? 'var(--ant-color-primary)' : 'var(--ant-color-text)',
                      }}
                      onClick={() => {
                        setCurrentAssistant({
                          _id: assistant._id,
                          _source: assistant._source,
                        });
                        setOpen(false);
                      }}
                    >
                      {assistant._source?.icon ? (
                        assistant._source.icon.startsWith("font_") ? (
                          <FontIcon
                            name={assistant._source.icon}
                            className="w-4 h-4"
                          />
                        ) : (
                          <img
                            src={assistant._source.icon}
                            className="w-4 h-4"
                            alt=""
                            onError={(e) => {
                              const el = e.currentTarget as HTMLImageElement;
                              el.style.display = "none";
                            }}
                          />
                        )
                      ) : null}
                      <div className="truncate flex-1">{name}</div>
                      {isActive && <Check className="w-4 h-4 ml-auto" style={{ color: 'var(--ant-color-primary)' }} />}
                    </button>
                  );
                })}
              </div>
            ) : (
              <div className="py-8 text-center text-sm" style={{ color: 'var(--ant-color-text-quaternary)' }}>
                {t("assistant_list.no_data")}
              </div>
            )}
          </div>
        </div>
      )}
      {open && (
        <div
          className="fixed inset-0 z-40"
          onClick={() => setOpen(false)}
        />
      )}
    </div>
  );
}

export function AssistantList(props: AssistantListProps) {
  return (
    <I18nextProvider i18n={i18n}>
      <InnerAssistantList {...props} />
    </I18nextProvider>
  );
}
