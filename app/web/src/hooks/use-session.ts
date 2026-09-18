import { useMutation, useQuery } from '@tanstack/react-query';
import { useCallback } from 'react';
import { logoutMutation } from '@/features/auth/api/mutations';
import { sessionQueryOptions } from '@/features/auth/api/queries';
import type { SessionUser, UserRole } from '@/features/auth/api/types';
import { useRouter } from '@/hooks/use-router';

/**
 * The replacement for Clerk's `useUser` / `useAuth`.
 *
 * There is no provider to wrap the app in: the session is one React Query
 * cache entry, so every caller of this hook reads the same state and only one
 * request is made however many components ask.
 *
 * The three states are deliberately distinct. `isPending` is "we do not know
 * yet", `user === null` is "definitely signed out", and they must not be
 * collapsed — a guard that treats "not known yet" as "signed out" bounces
 * every user to the sign-in page on a hard refresh.
 */
export function useSession() {
  const { data: user, isPending, isError, error } = useQuery(sessionQueryOptions);

  return {
    user: user ?? null,
    isLoaded: !isPending,
    isPending,
    isSignedIn: !!user,
    isError,
    error
  };
}

/**
 * Returns the signed-in user, asserting there is one.
 *
 * For components that only ever render inside ProtectedRoute, which is exactly
 * the condition that makes the assertion safe. It throws rather than returning
 * a nullable so the caller is not forced into optional chaining that would
 * quietly render an empty shell if the invariant ever broke.
 */
export function useSignedInUser(): SessionUser {
  const { user } = useSession();

  if (!user) {
    throw new Error('useSignedInUser was called outside an authenticated route');
  }

  return user;
}

/** True when the signed-in user holds the given role. */
export function useHasRole(role: UserRole): boolean {
  const { user } = useSession();

  return user?.role === role;
}

/**
 * Signs the user out and sends them to the sign-in page.
 *
 * Navigation happens on settle rather than on success for the same reason the
 * mutation clears the cache there: whatever the server said, the user asked to
 * leave.
 */
export function useSignOut() {
  const router = useRouter();
  const { mutate, isPending } = useMutation(logoutMutation);

  const signOut = useCallback(() => {
    mutate(undefined, {
      onSettled: () => router.push('/auth/sign-in')
    });
  }, [mutate, router]);

  return { signOut, isPending };
}
