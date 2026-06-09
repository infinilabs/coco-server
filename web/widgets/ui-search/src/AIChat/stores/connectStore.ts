import { create } from "zustand";
import { persist, subscribeWithSelector } from "zustand/middleware";
import { produce } from "immer";

type KeyArrayObject = {
  [key: string]: unknown[];
};

export interface Server {
  id?: string;
  name?: string;
  [key: string]: unknown;
}

export type IConnectStore = {
  serverList: Server[];
  setServerListSilently: (servers: Server[]) => void;
  currentService: Server;
  setCurrentService: (service: Server) => void;
  cloudSelectService: Server;
  setCloudSelectService: (service: Server) => void;
  connector_data: KeyArrayObject;
  setConnectorData: (connector_data: unknown[], key: string) => void;
  datasourceData: KeyArrayObject;
  setDatasourceData: (datasourceData: unknown[], key: string) => void;
  connectionTimeout?: number;
  setConnectionTimeout: (connectionTimeout?: number) => void;
  currentSessionId?: string;
  setCurrentSessionId: (currentSessionId?: string) => void;
  assistantList: unknown[];
  setAssistantList: (assistantList: unknown[]) => void;
  currentAssistant: unknown;
  setCurrentAssistant: (assistant: unknown) => void;
  querySourceTimeout?: number;
  setQuerySourceTimeout: (queryTimeout?: number) => void;
  visibleStartPage: boolean;
  setVisibleStartPage: (visibleStartPage: boolean) => void;
  allowSelfSignature: boolean;
  setAllowSelfSignature: (allowSelfSignature: boolean) => void;
  searchDelay: number;
  setSearchDelay: (searchDelay: number) => void;
  compactModeAutoCollapseDelay: number;
  setCompactModeAutoCollapseDelay: (
    compactModeAutoCollapseDelay: number
  ) => void;
};

export const useConnectStore = create<IConnectStore>()(
  subscribeWithSelector(
    persist(
      (set) => ({
        serverList: [],
        setServerListSilently: (serverList: Server[]) => {
          set(
            produce((draft: IConnectStore) => {
              draft.serverList = serverList;
            })
          );
        },
        currentService: {} as Server,
        setCurrentService: (server: Server) => {
          set(
            produce((draft: IConnectStore) => {
              draft.currentService = server;
            })
          );
        },
        cloudSelectService: {} as Server,
        setCloudSelectService: (server: Server) => {
          set(
            produce((draft: IConnectStore) => {
              draft.cloudSelectService = server;
            })
          );
        },
        connector_data: {},
        setConnectorData: (connector_data: unknown[], key: string) => {
          set(
            produce((draft: IConnectStore) => {
              draft.connector_data[key] = connector_data;
            })
          );
        },
        datasourceData: {},
        setDatasourceData: (datasourceData: unknown[], key: string) => {
          set(
            produce((draft: IConnectStore) => {
              draft.datasourceData[key] = datasourceData;
            })
          );
        },
        connectionTimeout: 120,
        setConnectionTimeout: (connectionTimeout) => {
          return set(() => ({ connectionTimeout }));
        },
        setCurrentSessionId(currentSessionId) {
          return set(() => ({ currentSessionId }));
        },
        assistantList: [],
        setAssistantList: (assistantList) => {
          return set(() => ({ assistantList }));
        },
        currentAssistant: null,
        setCurrentAssistant: (assistant: unknown) => {
          set(
            produce((draft: IConnectStore) => {
              draft.currentAssistant = assistant;
            })
          );
        },
        querySourceTimeout: 500,
        setQuerySourceTimeout: (queryTimeout) => {
          set(
            produce((draft) => {
              draft.querySourceTimeout = queryTimeout;
            })
          );
        },
        visibleStartPage: false,
        setVisibleStartPage: (visibleStartPage: boolean) => {
          return set(() => ({ visibleStartPage }));
        },
        allowSelfSignature: false,
        setAllowSelfSignature: (allowSelfSignature: boolean) => {
          return set(() => ({ allowSelfSignature }));
        },
        searchDelay: 300,
        setSearchDelay(searchDelay) {
          return set(() => ({ searchDelay }));
        },
        compactModeAutoCollapseDelay: 10,
        setCompactModeAutoCollapseDelay(compactModeAutoCollapseDelay) {
          return set(() => ({ compactModeAutoCollapseDelay }));
        },
      }),
      {
        name: "connect-store",
        partialize: (state) => ({
          currentAssistant: state.currentAssistant,
          querySourceTimeout: state.querySourceTimeout,
          allowSelfSignature: state.allowSelfSignature,
          searchDelay: state.searchDelay,
          compactModeAutoCollapseDelay: state.compactModeAutoCollapseDelay,
        }),
      }
    )
  )
);
