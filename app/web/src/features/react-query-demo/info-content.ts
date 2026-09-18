import type { InfobarContent } from '@/components/ui/infobar';

export const reactQueryInfoContent: InfobarContent = {
  title: 'React Query Pattern',
  sections: [
    {
      title: 'Suspense Query',
      description:
        'useSuspenseQuery fetches and suspends until the data resolves, so the Suspense fallback shows the skeleton rather than a spinner inside the card. The Next version also prefetched this on the server and hydrated the cache through HydrationBoundary; an SPA has no server pass, so only the client half of that pattern remains.',
      links: [
        {
          title: 'TanStack Query SSR Docs',
          url: 'https://tanstack.com/query/latest/docs/framework/react/guides/advanced-ssr'
        }
      ]
    },
    {
      title: 'Query Options',
      description:
        'Query keys and fetch functions are defined in a shared queryOptions() object, reused by every hook that touches the same data so keys never drift.',
      links: [
        {
          title: 'queryOptions API',
          url: 'https://tanstack.com/query/latest/docs/framework/react/reference/queryOptions'
        }
      ]
    },
    {
      title: 'Suspense Query',
      description:
        'useSuspenseQuery() integrates with React Suspense, so the nearest <Suspense> fallback renders while the query is in flight. Once the data is cached, revisiting the page is instant and the fallback only reappears when the cache goes stale.',
      links: []
    },
    {
      title: 'Optimistic Mutations',
      description:
        'Mutations use onMutate to optimistically update the cache before the request completes. On error, the previous state is rolled back. On settle, the query is invalidated to refetch fresh data.',
      links: [
        {
          title: 'Optimistic Updates Guide',
          url: 'https://tanstack.com/query/latest/docs/framework/react/guides/optimistic-updates'
        }
      ]
    }
  ]
};
