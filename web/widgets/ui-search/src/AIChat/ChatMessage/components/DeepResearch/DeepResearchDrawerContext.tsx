import { createContext, useCallback, useContext, useState, type FC, type ReactNode } from "react";
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
  openDrawer: (data: DeepResearchDrawerData) => void;
  updateDrawer: (data: Partial<DeepResearchDrawerData>) => void;
  closeDrawer: () => void;
  isOpen: boolean;
}

const DeepResearchDrawerContext = createContext<DeepResearchDrawerContextValue>({
  openDrawer: () => {},
  updateDrawer: () => {},
  closeDrawer: () => {},
  isOpen: false,
});

export const useDeepResearchDrawer = () => useContext(DeepResearchDrawerContext);

export const DeepResearchDrawerProvider: FC<{ children: ReactNode; isMobile?: boolean }> = ({ children, isMobile }) => {
  const [open, setOpen] = useState(false);
  const [drawerData, setDrawerData] = useState<DeepResearchDrawerData>({});
  const [revision, setRevision] = useState(0);

  const openDrawer = useCallback((data: DeepResearchDrawerData) => {
    setDrawerData(data);
    setRevision((r) => r + 1);
    setOpen(true);
  }, []);

  const updateDrawer = useCallback((data: Partial<DeepResearchDrawerData>) => {
    setDrawerData((prev) => ({ ...prev, ...data }));
  }, []);

  const closeDrawer = useCallback(() => {
    setOpen(false);
  }, []);

  return (
    <DeepResearchDrawerContext.Provider value={{ openDrawer, updateDrawer, closeDrawer, isOpen: open }}>
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
