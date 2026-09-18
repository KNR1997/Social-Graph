import { useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import * as React from 'react';

/**
 * Stand-in for next/navigation's useRouter, so copied components can keep
 * calling router.push/replace/back/refresh unchanged.
 *
 * `refresh()` has no server render to re-run here; the closest equivalent in
 * an SPA is invalidating the React Query cache, which is what every call site
 * in the Next version was actually after (re-read data after a mutation).
 */
export function useRouter() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  return React.useMemo(
    () => ({
      push: (href: string) => navigate(href),
      replace: (href: string) => navigate(href, { replace: true }),
      back: () => navigate(-1),
      forward: () => navigate(1),
      refresh: () => queryClient.invalidateQueries(),
      prefetch: (_href: string) => {}
    }),
    [navigate, queryClient]
  );
}
