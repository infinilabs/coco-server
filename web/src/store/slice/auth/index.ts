import { createSelector } from '@reduxjs/toolkit';

import { fetchGetUserInfo, fetchLogin } from '@/service/api';
import { localStg } from '@/utils/storage';

import type { AppThunk } from '../..';
import { createAppSlice } from '../../createAppSlice';
import { resetRouteStore } from '../route';
import { cacheTabs } from '../tab';

import { clearAuthStorage, getToken, getUserInfo } from './shared';

export const userInfo = {
  "id": "user123",
  "username": "InfiniLabs",
  "email": "InfiniLabs@example.com",
  "avatar": "https://example.com/images/avatar.jpg",
  "created": "2024-01-01T10:00:00Z",
  "updated": "2025-01-01T10:00:00Z",
  "roles": ["admin", "editor"],
  "preferences": {
    "theme": "light",
    "language": "en"
  }
}

const initialState = {
  token: getToken(),
  userInfo: getUserInfo()
};

export const authSlice = createAppSlice({
  initialState,
  name: 'auth',
  reducers: create => ({
    login: create.asyncThunk(
      async ({ password, userName }: { password: string; userName: string }) => {
        const u = 'Soybean';
        const p = '123456';
        const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjpbeyJ1c2VyTmFtZSI6IlNveWJlYW4ifV0sImlhdCI6MTY5ODQ4NDg2MywiZXhwIjoxNzMwMDQ0Nzk5LCJhdWQiOiJzb3liZWFuLWFkbWluIiwiaXNzIjoiU295YmVhbiIsInN1YiI6IlNveWJlYW4ifQ._w5wmPm6HVJc5fzkSrd_j-92d5PBRzWUfnrTF1bAmfk"
        localStg.set('token', token)
        localStg.set('refreshToken', token)
        localStg.set('userInfo', userInfo);
        return {
          token,
          userInfo: userInfo
        };
        // const { data: loginToken, error } = await fetchLogin(u, p);
        // // 1. stored in the localStorage, the later requests need it in headers
        // if (!error) {
        //   localStg.set('token', loginToken.token);
        //   localStg.set('refreshToken', loginToken.refreshToken);

        //   const { data: info, error: userInfoError } = await fetchGetUserInfo();

        //   if (!userInfoError) {
        //     // 2. store user info
        //     localStg.set('userInfo', info);
        //     return {
        //       token: loginToken.token,
        //       userInfo: info
        //     };
        //   }
        // }

        return false;
      },

      {
        fulfilled: (state, { payload }) => {
          if (payload) {
            state.token = payload.token;
            state.userInfo = payload.userInfo;
          }
        }
      }
    ),
    resetAuth: create.reducer(() => initialState)
  }),
  selectors: {
    selectToken: auth => auth.token,
    selectUserInfo: auth => auth.userInfo
  }
});
export const { selectToken, selectUserInfo } = authSlice.selectors;
export const { login, resetAuth } = authSlice.actions;
// We can also write thunks by hand, which may contain both sync and async logic.
// Here's an example of conditionally dispatching actions based on current state.
export const getUerName = (): AppThunk<string> => (_, getState) => {
  const pass = selectToken(getState());

  return pass ? selectUserInfo(getState()).username : '';
};

/** is super role in static route */

export const isStaticSuper = (): AppThunk<boolean> => (_, getState) => {
  const { roles } = selectUserInfo(getState());

  const { VITE_AUTH_ROUTE_MODE, VITE_STATIC_SUPER_ROLE } = import.meta.env;
  return VITE_AUTH_ROUTE_MODE === 'static' && roles.includes(VITE_STATIC_SUPER_ROLE);
};

/** Reset auth store */
export const resetStore = (): AppThunk => dispatch => {
  clearAuthStorage();

  dispatch(resetAuth());

  dispatch(resetRouteStore());

  dispatch(cacheTabs());
};

/** Is login */
export const getIsLogin = createSelector([selectToken], token => Boolean(token));
