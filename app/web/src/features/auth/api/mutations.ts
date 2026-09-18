import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { authKeys } from './queries';
import { changePassword, login, logout, register, updateProfile } from './service';
import type {
  ChangePasswordPayload,
  LoginPayload,
  RegisterPayload,
  SessionResponse
} from './types';

/**
 * Seeds the session cache from a sign-in response.
 *
 * Writing the user straight into the cache rather than invalidating avoids a
 * redundant round trip and, more importantly, avoids the flicker where the
 * route guard briefly sees "no session" between navigating and the refetch
 * landing.
 */
function adoptSession(response: SessionResponse) {
  getQueryClient().setQueryData(authKeys.session(), response.user);
}

export const loginMutation = mutationOptions({
  mutationFn: (payload: LoginPayload) => login(payload),
  onSuccess: adoptSession
});

export const registerMutation = mutationOptions({
  mutationFn: (payload: RegisterPayload) => register(payload),
  onSuccess: adoptSession
});

export const logoutMutation = mutationOptions({
  mutationFn: () => logout(),
  /**
   * onSettled, not onSuccess: if sign-out fails the safest local state is still
   * "signed out". The cookie may already be gone, and leaving a stale user in
   * the cache would show a signed-in shell that 401s on every request.
   *
   * clear() rather than a targeted invalidate, so no other feature's cached
   * data outlives the session that was allowed to read it.
   */
  onSettled: () => {
    getQueryClient().clear();
  }
});

export const updateProfileMutation = mutationOptions({
  mutationFn: (name: string) => updateProfile(name),
  onSuccess: (user) => {
    getQueryClient().setQueryData(authKeys.session(), user);
  }
});

export const changePasswordMutation = mutationOptions({
  mutationFn: (payload: ChangePasswordPayload) => changePassword(payload),
  onSuccess: adoptSession
});
