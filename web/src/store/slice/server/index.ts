import { createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

import { AppThunk } from '@/store';
import { getRootRouteIfSearch } from './shared';
import { handleUpdateRootRouteRedirect, setRouteHome } from '../route';
import { isPlainObject } from 'lodash';

interface InitialStateType {
  applicationSetting: any;
  providerInfo: any;
  defaultModel: any;
  defaultModelTips: boolean;
}

const initialState: InitialStateType = {
  applicationSetting: {},
  providerInfo: {},
  defaultModel: {},
  defaultModelTips: false
};

export const serverSlice = createSlice({
  initialState,
  name: 'server',
  reducers: {
    setApplicationSetting(state, { payload }: PayloadAction<any>){
      state.applicationSetting = payload;
    },
    setProviderInfo(state, { payload }: PayloadAction<any>) {
      state.providerInfo = payload;
    },
    setDefaultModel(state, { payload }: PayloadAction<any>) {
      state.defaultModel = payload;
      state.defaultModelTips = ![
        payload?.language_model,
        payload?.vision_model,
        payload?.embedding_model
      ].every((model) => {
        return isPlainObject(model) && !!model.id && !!model.provider_id;
      });
    },
    setDefaultModelTips(state, { payload }: PayloadAction<any>) {
      state.defaultModelTips = payload;
    }
  },
  selectors: {
    getApplicationSetting: app => app.applicationSetting,
    getProviderInfo: app => app.providerInfo,
    getServer: app => app.providerInfo?.endpoint || `${window.location.origin}${window.location.pathname}`,
    getDefaultModel: app => app.defaultModel,
    getDefaultModelTips: app => app.defaultModelTips
  }
});
// Action creators are generated for each case reducer function.
export const {
  setApplicationSetting,
  setProviderInfo,
  setDefaultModel,
  setDefaultModelTips,
} = serverSlice.actions;

// Selectors returned by `slice.selectors` take the root state as their first argument.
export const {
  getApplicationSetting,
  getProviderInfo,
  getServer,
  getDefaultModel,
  getDefaultModelTips,
} = serverSlice.selectors;

export const updateRootRouteIfSearch = (applicationSetting: any): AppThunk => async dispatch => {
  const rootRoute = getRootRouteIfSearch(applicationSetting) as any
  dispatch(setRouteHome(rootRoute));
  handleUpdateRootRouteRedirect(rootRoute)
};