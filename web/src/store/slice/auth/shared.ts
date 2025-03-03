import { localStg } from '@/utils/storage';
/** Get token */
export function getToken() {
  return localStg.get('token') || '';
}

/** Get user info */
export function getUserInfo() {

  const userInfo = localStg.get('userInfo') || {
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

  return userInfo;

}

/** Clear auth storage */
export function clearAuthStorage() {
  localStg.remove('token');
  localStg.remove('refreshToken');
  localStg.remove('userInfo');
}
