import { queryOptions } from '@tanstack/react-query';
import { getSession } from './service';

export const authKeys = {
  all: ['auth'] as const,
  session: () => [...authKeys.all, 'session'] as const
};

/**
 * The session query. Every part of the UI that needs to know who is signed in
 * reads this one cache entry, so there is a single source of truth and a single
 * request on load.
 *
 * `retry: false` matters: a signed-out visitor resolves to null rather than
 * erroring, but a genuinely failing request should surface immediately instead
 * of leaving the app on a spinner through three backoff attempts.
 */
export const sessionQueryOptions = queryOptions({
  queryKey: authKeys.session(),
  queryFn: ({ signal }) => getSession(signal),
  retry: false,
  // The server is the authority on whether the session is still alive, and the
  // cost of asking is one cheap request. Re-check when the tab regains focus so
  // a session that expired or was revoked elsewhere is noticed.
  staleTime: 30 * 1000,
  refetchOnWindowFocus: true
});
