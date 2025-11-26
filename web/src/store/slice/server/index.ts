import { createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

import { localStg } from '@/utils/storage';
import { AppThunk } from '@/store';
import { getRootRouteIfSearch } from './shared';
import { handleUpdateRootRouteRedirect, setRouteHome } from '../route';

interface InitialStateType {
  providerInfo: any;
}

const initialState: InitialStateType = {
  providerInfo: {}
};

export const serverSlice = createSlice({
  initialState,
  name: 'server',
  reducers: {
    setProviderInfo(state, { payload }: PayloadAction<any>) {
      state.providerInfo = payload;
      localStg.set('providerInfo', payload);
    },
  },
  selectors: {
    getProviderInfo: app => app.providerInfo,
    getServer: app => app.providerInfo?.endpoint || `${window.location.origin}${window.location.pathname}`
  }
});
// Action creators are generated for each case reducer function.
export const {
  setProviderInfo,
} = serverSlice.actions;

// Selectors returned by `slice.selectors` take the root state as their first argument.
export const {
  getProviderInfo,
  getServer
} = serverSlice.selectors;

export const updateRootRouteIfSearch = (providerInfo: any): AppThunk => async dispatch => {
  const rootRoute = getRootRouteIfSearch(providerInfo) as any
  dispatch(setRouteHome(rootRoute));
  handleUpdateRootRouteRedirect(rootRoute)
};