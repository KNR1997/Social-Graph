import type React from 'react';
import QueryProvider from './query-provider';

/**
 * The application provider stack.
 *
 * Session auth needs no provider of its own. Clerk required one because it
 * held the session in its own client-side store; here the session is an
 * HttpOnly cookie the browser manages and one React Query cache entry
 * (`sessionQueryOptions`) that reflects it, so QueryProvider is the only thing
 * that has to be above the tree.
 */
export default function Providers({ children }: { children: React.ReactNode }) {
  return <QueryProvider>{children}</QueryProvider>;
}
