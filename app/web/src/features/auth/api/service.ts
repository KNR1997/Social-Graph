// ============================================================
// Auth Service — Data Access Layer
// ============================================================
// Unlike the other features in this project, this one is not mock-backed:
// authentication has to be real to be worth anything. Every function here
// calls the Go API at /v1/auth, and the session lives in an HttpOnly cookie
// that `apiFetch` opts into with `credentials: 'include'`.
//
// The cookie is why there is no token to store, no Authorization header to
// build, and no "log the user back in on refresh" logic: the browser presents
// the session on every request until it expires or the server deletes it.
// ============================================================

import { ApiError, apiFetch } from '@/lib/api';
import type {
  ChangePasswordPayload,
  LoginPayload,
  RegisterPayload,
  SessionResponse,
  SessionUser
} from './types';

/**
 * Returns the signed-in user, or null when there is no session.
 *
 * A 401 is an answer, not a failure: it is how the API says "nobody is signed
 * in". Letting it throw would put the session query into an error state and
 * make "signed out" indistinguishable from "the API is down", which is a
 * distinction the route guard depends on.
 */
export async function getSession(signal?: AbortSignal): Promise<SessionUser | null> {
  try {
    return await apiFetch<SessionUser>('/v1/auth/me', { signal });
  } catch (error) {
    if (error instanceof ApiError && error.isUnauthorized) {
      return null;
    }

    throw error;
  }
}

export function login(payload: LoginPayload): Promise<SessionResponse> {
  return apiFetch<SessionResponse>('/v1/auth/login', { method: 'POST', body: payload });
}

export function register(payload: RegisterPayload): Promise<SessionResponse> {
  return apiFetch<SessionResponse>('/v1/auth/register', { method: 'POST', body: payload });
}

export function logout(): Promise<void> {
  return apiFetch<void>('/v1/auth/logout', { method: 'POST' });
}

export function updateProfile(name: string): Promise<SessionUser> {
  return apiFetch<SessionUser>('/v1/auth/me', { method: 'PATCH', body: { name } });
}

/**
 * Changes the password. The API destroys every other session for this user and
 * re-issues this browser's cookie, so the caller stays signed in and everyone
 * else is evicted.
 */
export function changePassword(payload: ChangePasswordPayload): Promise<SessionResponse> {
  return apiFetch<SessionResponse>('/v1/auth/me/password', { method: 'POST', body: payload });
}
