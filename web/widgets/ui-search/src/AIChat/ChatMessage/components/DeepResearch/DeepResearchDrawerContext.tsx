import { createContext, useCallback, useContext, useEffect, useRef, useState, type FC, type ReactNode } from "react";
import { type TFunction } from "i18next";

import { DeepResearchDrawer } from "./DeepResearchDrawer";
import type { StepItem, StepStatus, StepSearchHit } from "./ResearchStepsContent";
import type { ResearchReportData } from "./ResearchReportContent";

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
}

interface DeepResearchDrawerContextValue {
  openDrawer: (data: DeepResearchDrawerData, sourceId?: string) => void;
  updateDrawer: (data: Partial<DeepResearchDrawerData>, sourceId?: string) => void;
  closeDrawer: () => void;
  isOpen: boolean;
  activeSourceId: string | undefined;
}

const DeepResearchDrawerContext = createContext<DeepResearchDrawerContextValue>({
  openDrawer: () => {},
  updateDrawer: () => {},
  closeDrawer: () => {},
  isOpen: false,
  activeSourceId: undefined,
});

export const useDeepResearchDrawer = () => useContext(DeepResearchDrawerContext);

export const DeepResearchDrawerProvider: FC<{ children: ReactNode; isMobile?: boolean; chatId?: string }> = ({ children, isMobile, chatId }) => {
  const [open, setOpen] = useState(false);
  const [drawerData, setDrawerData] = useState<DeepResearchDrawerData>({});
  const [revision, setRevision] = useState(0);
  const activeSourceIdRef = useRef<string | undefined>(undefined);
  const [activeSourceId, setActiveSourceId] = useState<string | undefined>(undefined);

  const prevChatIdRef = useRef(chatId);
  useEffect(() => {
    if (prevChatIdRef.current !== chatId) {
      prevChatIdRef.current = chatId;
      setOpen(false);
      activeSourceIdRef.current = undefined;
      setActiveSourceId(undefined);
    }
  }, [chatId]);

  const openDrawer = useCallback((data: DeepResearchDrawerData, sourceId?: string) => {
    setDrawerData(data);
    setRevision((r) => r + 1);
    setOpen(true);
    activeSourceIdRef.current = sourceId;
    setActiveSourceId(sourceId);
  }, []);

  const updateDrawer = useCallback((data: Partial<DeepResearchDrawerData>, sourceId?: string) => {
    // Only allow updates from the instance that opened the drawer
    if (sourceId && activeSourceIdRef.current && sourceId !== activeSourceIdRef.current) {
      return;
    }
    setDrawerData((prev) => ({ ...prev, ...data }));
  }, []);

  const closeDrawer = useCallback(() => {
    setOpen(false);
    activeSourceIdRef.current = undefined;
    setActiveSourceId(undefined);
  }, []);

  return (
    <DeepResearchDrawerContext.Provider value={{ openDrawer, updateDrawer, closeDrawer, isOpen: open, activeSourceId }}>
      {children}
      <DeepResearchDrawer
        open={open}
        onClose={closeDrawer}
        revision={revision}
        isMobile={isMobile}
        {...drawerData}
      />
    </DeepResearchDrawerContext.Provider>
  );
};
