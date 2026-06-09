import { useState, useCallback } from "react";

import type { IChunkData } from "../types/chat";

export default function useMessageChunkData() {
  const [query_intent, setQuery_intent] = useState<IChunkData>();
  const [tools, setTools] = useState<IChunkData>();
  const [fetch_source, setFetch_source] = useState<IChunkData>();
  const [pick_source, setPick_source] = useState<IChunkData>();
  const [deep_read, setDeep_read] = useState<IChunkData>();
  const [think, setThink] = useState<IChunkData>();
  const [response, setResponse] = useState<IChunkData>();

  // Deep Research State
  const [deepResearchPlans, setDeepResearchPlans] = useState<string[]>([]);
  const [deepResearchCurrentStepIndex, setDeepResearchCurrentStepIndex] = useState<number>(-1);
  const [deepResearchQuery, setDeepResearchQuery] = useState<string>("");
  const [deepResearchResultCount, setDeepResearchResultCount] = useState<number | undefined>(undefined);
  const [deepResearchResearcherStarted, setDeepResearchResearcherStarted] = useState(false);
  const [deepResearchReporterStarted, setDeepResearchReporterStarted] = useState(false);
  const [deepResearchReporterFinished, setDeepResearchReporterFinished] = useState(false);
  const [deepResearchReportData, setDeepResearchReportData] = useState<any | undefined>(undefined);
  const [deepResearchSearchMap, setDeepResearchSearchMap] = useState<Record<string, any>>({});

  const resetDeepResearchState = useCallback(() => {
    setDeepResearchPlans([]);
    setDeepResearchCurrentStepIndex(-1);
    setDeepResearchQuery("");
    setDeepResearchResultCount(undefined);
    setDeepResearchResearcherStarted(false);
    setDeepResearchReporterStarted(false);
    setDeepResearchReporterFinished(false);
    setDeepResearchReportData(undefined);
    setDeepResearchSearchMap({});
  }, []);

  const handlers = {
    deal_query_intent: useCallback((data: IChunkData) => {
      setQuery_intent((prev: IChunkData | undefined): IChunkData => {
        if (!prev) return data;
        return {
          ...prev,
          message_chunk: data.message_chunk ?? prev.message_chunk,
        };
      });
    }, []),
    deal_tools: useCallback((data: IChunkData) => {
      setTools((prev: IChunkData | undefined): IChunkData => {
        if (!prev) return data;
        return {
          ...prev,
          message_chunk: data.message_chunk ?? prev.message_chunk,
        };
      });
    }, []),
    deal_fetch_source: useCallback((data: IChunkData) => {
      setFetch_source((prev: IChunkData | undefined): IChunkData => {
        if (!prev) return data;
        return {
          ...prev,
          message_chunk: data.message_chunk ?? prev.message_chunk,
        };
      });
    }, []),
    deal_pick_source: useCallback((data: IChunkData) => {
      setPick_source((prev: IChunkData | undefined): IChunkData => {
        if (!prev) return data;
        return {
          ...prev,
          message_chunk: data.message_chunk ?? prev.message_chunk,
        };
      });
    }, []),
    deal_deep_read: useCallback((data: IChunkData) => {
      setDeep_read((prev: IChunkData | undefined): IChunkData => {
        if (!prev) return data;
        return {
          ...prev,
          message_chunk:
            (prev.message_chunk ?? "") +
            (prev.message_chunk && data.message_chunk ? "&" : "") +
            (data.message_chunk ?? ""),
        };
      });
    }, []),
    deal_think: useCallback((data: IChunkData) => {
      setThink((prev: IChunkData | undefined): IChunkData => {
        if (!prev) return data;
        return {
          ...prev,
          message_chunk: data.message_chunk ?? prev.message_chunk,
        };
      });
    }, []),
    deal_response: useCallback((data: IChunkData) => {
      setResponse((prev: IChunkData | undefined): IChunkData => {
        if (!prev) return data;
        return {
          ...prev,
          message_chunk: data.message_chunk ?? prev.message_chunk,
        };
      });
    }, []),
    deal_deep_research: useCallback((chunkData: IChunkData) => {
      if (chunkData.chunk_type === "research_planner_start") {
        resetDeepResearchState();
        return;
      }
  
      if (chunkData.chunk_type === "research_planner_end") {
        if (typeof chunkData.message_chunk === "string") {
          try {
            const payload = JSON.parse(chunkData.message_chunk);
            if (Array.isArray(payload)) {
              const plans = payload.map((item) => String(item));
              setDeepResearchPlans(plans);
              setDeepResearchCurrentStepIndex(plans.length > 0 ? 0 : -1);
            }
          } catch (error) {
            console.error(error);
          }
        }
        return;
      }
  
      if (chunkData.chunk_type === "research_researcher_start") {
        if (typeof chunkData.message_chunk === "string" && chunkData.message_chunk) {
          try {
            const payload = JSON.parse(chunkData.message_chunk);
            const planText = typeof payload?.plan === "string" ? payload.plan : "";
            if (planText) {
              setDeepResearchResearcherStarted(true);
              setDeepResearchCurrentStepIndex((prevIndex) => {
                // Accessing state directly here might be stale if not careful, 
                // but React setters with callback get current state.
                // However, deepResearchPlans is another state.
                // We should pass deepResearchPlans to this callback or use refs?
                // `deal_deep_research` is created with `useCallback`.
                // If we don't include `deepResearchPlans` in dependency array, it's stale.
                // If we do, `handlers` changes often.
                // ChatMessage implementation used `deepResearchPlans` from component scope.
                
                // Since we are inside a hook, we can't easily access the *current* deepResearchPlans inside this callback unless we add it to dependency.
                // But let's look at how ChatMessage did it.
                // ChatMessage component re-renders on every state change, so `handleDeepResearchChunk` is recreated (it wasn't wrapped in useCallback there).
                
                // Here we wrap in useCallback. We MUST add dependencies.
                // Or better, use functional updates where possible, but here we need `deepResearchPlans` to find index.
                return prevIndex; // Placeholder, see logic below
              });
            }
          } catch (error) {
            console.error(error);
          }
        }
        return;
      }
      
      // ... (rest of logic)
    }, [resetDeepResearchState]), // dependencies will be tricky
  };

  // Re-implementing deal_deep_research to handle dependencies correctly
  // or removing useCallback for it if performance allows (it's called during streaming).
  
  const deal_deep_research = (chunkData: IChunkData) => {
    if (chunkData.chunk_type === "research_planner_start") {
      resetDeepResearchState();
      return;
    }

    if (chunkData.chunk_type === "research_planner_end") {
      if (typeof chunkData.message_chunk === "string") {
        try {
          const payload = JSON.parse(chunkData.message_chunk);
          if (Array.isArray(payload)) {
            const plans = payload.map((item) => String(item));
            setDeepResearchPlans(plans);
            setDeepResearchCurrentStepIndex(plans.length > 0 ? 0 : -1);
          }
        } catch (error) {
          console.error(error);
        }
      }
      return;
    }

    if (chunkData.chunk_type === "research_researcher_start") {
      if (typeof chunkData.message_chunk === "string" && chunkData.message_chunk) {
        try {
          const payload = JSON.parse(chunkData.message_chunk);
          const planText = typeof payload?.plan === "string" ? payload.plan : "";
          if (planText) {
            setDeepResearchResearcherStarted(true);
            setDeepResearchCurrentStepIndex((prevIndex) => {
              const index = deepResearchPlans.findIndex(
                (title) => title === planText
              );
              if (index !== -1) return index;
              if (prevIndex >= 0) return prevIndex;
              return 0;
            });
          }
        } catch (error) {
          console.error(error);
        }
      }
      return;
    }

    if (chunkData.chunk_type === "research_researcher_step_start") {
      if (typeof chunkData.message_chunk === "string" && chunkData.message_chunk) {
        try {
          const payload = JSON.parse(chunkData.message_chunk);
          const planText = typeof payload?.plan === "string" ? payload.plan : "";
          const stepQuery = payload?.step?.payload?.query;
          if (typeof stepQuery === "string") {
            setDeepResearchQuery(stepQuery);
          }
          setDeepResearchResultCount(undefined);
          if (planText && typeof stepQuery === "string") {
            setDeepResearchSearchMap((prev) => {
              const prevInfo = prev[planText] ?? {};
              return {
                ...prev,
                [planText]: {
                  ...prevInfo,
                  query: stepQuery,
                },
              };
            });
          }
        } catch (error) {
          console.error(error);
        }
      }
      return;
    }

    if (chunkData.chunk_type === "research_researcher_step_end") {
      if (typeof chunkData.message_chunk === "string" && chunkData.message_chunk) {
        try {
          const payload = JSON.parse(chunkData.message_chunk);
          const planText = typeof payload?.plan === "string" ? payload.plan : "";
          const hits = payload?.step?.payload?.hits;
          if (Array.isArray(hits)) {
            setDeepResearchResultCount(hits.length);
            if (planText) {
              setDeepResearchSearchMap((prev) => {
                const prevInfo = prev[planText] ?? {};
                return {
                  ...prev,
                  [planText]: {
                    ...prevInfo,
                    resultCount: hits.length,
                    hits: hits,
                  },
                };
              });
            }
          }
        } catch (error) {
          console.error(error);
        }
      }
      return;
    }

    if (chunkData.chunk_type === "research_researcher_end") {
      setDeepResearchQuery("");
      return;
    }

    if (chunkData.chunk_type === "research_reporter_start") {
      setDeepResearchReporterStarted(true);
      return;
    }

    if (chunkData.chunk_type === "research_reporter_end") {
      setDeepResearchReporterStarted(true);
      setDeepResearchReporterFinished(true);
      if (typeof chunkData.message_chunk === "string" && chunkData.message_chunk) {
        try {
          const payload = JSON.parse(chunkData.message_chunk);
          setDeepResearchReportData(payload);
        } catch (error) {
          console.error(error);
        }
      }
    }
  };

  const clearAllChunkData = () => {
    return new Promise<void>((resolve) => {
      setQuery_intent(undefined);
      setTools(undefined);
      setFetch_source(undefined);
      setPick_source(undefined);
      setDeep_read(undefined);
      setThink(undefined);
      setResponse(undefined);
      resetDeepResearchState();
      setTimeout(resolve, 0);
    });
  };

  return {
    data: {
      query_intent,
      tools,
      fetch_source,
      pick_source,
      deep_read,
      think,
      response,
      // Deep Research Data
      deepResearchPlans,
      deepResearchCurrentStepIndex,
      deepResearchQuery,
      deepResearchResultCount,
      deepResearchResearcherStarted,
      deepResearchReporterStarted,
      deepResearchReporterFinished,
      deepResearchReportData,
      deepResearchSearchMap,
    },
    handlers: {
        ...handlers,
        deal_deep_research: useCallback(deal_deep_research, [
            deepResearchPlans, // Add dependency
            resetDeepResearchState,
            // Add other dependencies if needed, but setters are stable
        ])
    },
    clearAllChunkData,
    resetDeepResearchState,
  };
}