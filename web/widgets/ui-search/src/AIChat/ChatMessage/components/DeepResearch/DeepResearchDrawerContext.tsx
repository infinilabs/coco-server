import { createContext, useCallback, useContext, useEffect, useRef, useState, type FC, type ReactNode } from "react";
import { type TFunction } from "i18next";

import { DeepResearchDrawer } from "./DeepResearchDrawer";
import type { StepItem, StepStatus, StepSearchHit } from "./ResearchStepsContent";
import type { ResearchEndChunk, ResearchReportData } from "./ResearchReportContent";
import { useChatStore } from "../../../stores/chatStore";

export interface DeepResearchDrawerData {
  defaultActiveTab?: string;
  steps?: StepItem[];
  plannerStatus?: StepStatus;
  executionStatus?: StepStatus;
  reportStatus?: StepStatus;
  reportData?: ResearchReportData;
  reportContent?: string;
  searchHits?: StepSearchHit[];
  formatUrl?: (data: any) => string;
  requestHeaders?: Record<string, string>;
  theme?: "light" | "dark";
  isMobile?: boolean;
  showReportOnly?: boolean;
  t?: TFunction;
  isEnd?: boolean;
  endChunk?: ResearchEndChunk;
}

interface DeepResearchDrawerContextValue {
  openDrawer: (data: DeepResearchDrawerData, sourceIds?: string | string[]) => void;
  updateDrawer: (data: Partial<DeepResearchDrawerData>, sourceIds?: string | string[]) => void;
  closeDrawer: () => void;
  isOpen: boolean;
  revision: number;
}

const DeepResearchDrawerContext = createContext<DeepResearchDrawerContextValue>({
  openDrawer: () => {},
  updateDrawer: () => {},
  closeDrawer: () => {},
  isOpen: false,
  revision: 0,
});

export const useDeepResearchDrawer = () => useContext(DeepResearchDrawerContext);

const normalizeSourceIds = (sourceIds?: string | string[]) => {
  return (Array.isArray(sourceIds) ? sourceIds : [sourceIds]).filter(Boolean) as string[];
};

export const DeepResearchDrawerProvider: FC<{ children: ReactNode; isMobile?: boolean; chatId?: string; theme?: string }> = ({ children, isMobile, chatId, theme }) => {
  const [open, setOpen] = useState(false);
  const [drawerData, setDrawerData] = useState<DeepResearchDrawerData>({});
  const [revision, setRevision] = useState(0);
  const activeSourceIdsRef = useRef<Set<string>>(new Set());
  const setDeepResearchDrawerOpen = useChatStore((state) => state.setDeepResearchDrawerOpen);

  const prevChatIdRef = useRef(chatId);
  useEffect(() => {
    if (prevChatIdRef.current !== chatId) {
      prevChatIdRef.current = chatId;
      setOpen(false);
    }
  }, [chatId]);

  useEffect(() => {
    setDeepResearchDrawerOpen(open && !isMobile);
  }, [open, isMobile, setDeepResearchDrawerOpen]);

  const openDrawer = useCallback((data: DeepResearchDrawerData, sourceIds?: string | string[]) => {
    activeSourceIdsRef.current = new Set(normalizeSourceIds(sourceIds));
    setDrawerData(data);
    setRevision((r) => r + 1);
    setOpen(true);
  }, []);

  const updateDrawer = useCallback((data: Partial<DeepResearchDrawerData>, sourceIds?: string | string[]) => {
    const nextSourceIds = normalizeSourceIds(sourceIds);
    const activeSourceIds = activeSourceIdsRef.current;
    if (
      nextSourceIds.length > 0 &&
      activeSourceIds.size > 0 &&
      !nextSourceIds.some((sourceId) => activeSourceIds.has(sourceId))
    ) {
      return;
    }
    nextSourceIds.forEach((sourceId) => activeSourceIds.add(sourceId));
    setDrawerData((prev) => ({ ...prev, ...data }));
  }, []);

  const closeDrawer = useCallback(() => {
    setOpen(false);
  }, []);

  return (
    <DeepResearchDrawerContext.Provider value={{ openDrawer, updateDrawer, closeDrawer, isOpen: open, revision }}>
      {children}
      <DeepResearchDrawer
        open={open}
        onClose={closeDrawer}
        revision={revision}
        isMobile={isMobile}
        theme={theme}
        {...drawerData}
      />
    </DeepResearchDrawerContext.Provider>
  );
};
