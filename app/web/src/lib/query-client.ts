import { QueryClient } from '@tanstack/react-query';

// The Next version branched on `isServer` to build a fresh QueryClient per
// server request (so requests never shared a cache) and a singleton in the
// browser. An SPA only ever runs in the browser, so the singleton is all
// that's left.
let queryClient: QueryClient | undefined;

export function getQueryClient() {
  if (!queryClient) {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          staleTime: 60 * 1000
        }
      }
    });
  }
  return queryClient;
}
