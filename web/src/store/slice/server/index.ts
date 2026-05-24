import { createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

import { localStg } from '@/utils/storage';
import { AppThunk } from '@/store';
import { getRootRouteIfSearch } from './shared';
import { handleUpdateRootRouteRedirect, setRouteHome } from '../route';
import { isEmpty } from 'lodash';

interface InitialStateType {
  providerInfo: any;
  defaultModel: any;
  defaultModelTips: boolean;
}

const initialState: InitialStateType = {
  providerInfo: {},
  defaultModel: {},
  defaultModelTips: false
};

export const serverSlice = createSlice({
  initialState,
  name: 'server',
  reducers: {
    setProviderInfo(state, { payload }: PayloadAction<any>) {
      state.providerInfo = payload;
      localStg.set('providerInfo', payload);
    },
    setServer(state, { payload }: PayloadAction<string>) {
      if (state.providerInfo) {
        state.providerInfo.endpoint = payload;
        localStg.set('providerInfo', state.providerInfo);
      }
    },
    setDefaultModel(state, { payload }: PayloadAction<any>) {
      state.defaultModel = payload;
      state.defaultModelTips = isEmpty(payload);
    },
    setDefaultModelTips(state, { payload }: PayloadAction<any>) {
      state.defaultModelTips = payload;
    }
  },
  selectors: {
    getProviderInfo: app => app.providerInfo,
    getServer: app => app.providerInfo?.endpoint || `${window.location.origin}${window.location.pathname}`,
    getDefaultModel: app => app.defaultModel,
    getDefaultModelTips: app => app.defaultModelTips
  }
});
// Action creators are generated for each case reducer function.
export const {
  setProviderInfo,
  setServer,
  setDefaultModel,
  setDefaultModelTips,
} = serverSlice.actions;

// Selectors returned by `slice.selectors` take the root state as their first argument.
export const {
  getProviderInfo,
  getServer,
  getDefaultModel,
  getDefaultModelTips,
} = serverSlice.selectors;

export const updateRootRouteIfSearch = (providerInfo: any): AppThunk => async dispatch => {
  const rootRoute = getRootRouteIfSearch(providerInfo) as any
  dispatch(setRouteHome(rootRoute));
  handleUpdateRootRouteRedirect(rootRoute)
};