import { createSelector } from "@reduxjs/toolkit";

import { fetchGetUserInfo, fetchLogin } from "@/service/api";
import { localStg } from "@/utils/storage";

import type { AppThunk } from "../..";
import { createAppSlice } from "../../createAppSlice";
import { resetRouteStore } from "../route";
import { cacheTabs } from "../tab";

import { clearAuthStorage, getToken, getUserInfo } from "./shared";

const initialState = {
  token: getToken(),
  userInfo: getUserInfo(),
};

export const authSlice = createAppSlice({
  initialState,
  name: "auth",
  reducers: (create) => ({
    login: create.asyncThunk(
      async (params: { password: string; email: string }) => {
        const { data, error } = await fetchLogin(params);
        // 1. stored in the localStorage, the later requests need it in headers
        if (!error) {
          const { data: userInfo, error: userInfoError } = await fetchGetUserInfo();

          if (!userInfoError) {
            // 2. store user info
            localStg.set("userInfo", userInfo);
            return {
              token: data.access_token,
              userInfo: userInfo,
            };
          }
        }

        return false;
      },

      {
        fulfilled: (state, { payload }) => {
          if (payload) {
            state.token = payload.token;
            state.userInfo = payload.userInfo;
          }
        },
      },
    ),
    resetAuth: create.reducer(() => ({
      token: getToken(),
      userInfo: getUserInfo(),
    })),
  }),
  selectors: {
    selectToken: (auth) => auth.token,
    selectUserInfo: (auth) => auth.userInfo,
  },
});
export const { selectToken, selectUserInfo } = authSlice.selectors;
export const { login, resetAuth } = authSlice.actions;
// We can also write thunks by hand, which may contain both sync and async logic.
// Here's an example of conditionally dispatching actions based on current state.
export const getUerName = (): AppThunk<string> => (_, getState) => {
  const pass = selectToken(getState());

  return pass ? selectUserInfo(getState())?.name : "";
};

/** is super role in static route */

export const isStaticSuper = (): AppThunk<boolean> => (_, getState) => {
  const { roles = [] } = selectUserInfo(getState()) || {};

  const { VITE_AUTH_ROUTE_MODE, VITE_STATIC_SUPER_ROLE } = import.meta.env;
  return (
    VITE_AUTH_ROUTE_MODE === "static" && roles.includes(VITE_STATIC_SUPER_ROLE)
  );
};

/** Reset auth store */
export const resetStore = (): AppThunk => (dispatch) => {
  clearAuthStorage();

  dispatch(resetAuth());

  dispatch(resetRouteStore());

  dispatch(cacheTabs());
};

/** Is login */
export const getIsLogin = createSelector([selectUserInfo], (userInfo) =>
  Boolean(userInfo),
);
