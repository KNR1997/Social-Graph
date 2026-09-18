import type { InfobarContent } from '@/components/ui/infobar';

export const usersInfoContent: InfobarContent = {
  title: 'Users — React Query + nuqs Pattern',
  sections: [
    {
      title: 'Overview',
      description:
        'This page demonstrates client-side data fetching with React Query combined with nuqs URL search params. The Products page follows the identical pattern; both share the same DataTable, useDataTable hook, and nuqs URL state.',
      links: [
        {
          title: 'TanStack Query SSR Docs',
          url: 'https://tanstack.com/query/latest/docs/framework/react/guides/advanced-ssr'
        }
      ]
    },
    {
      title: 'Suspense Query + URL-derived Filters',
      description:
        'The table reads the search params with nuqs useQueryStates, builds the filter object, and passes it to useSuspenseQuery. Because the filters are part of the query key, changing the URL is what triggers the refetch. The Next version additionally prefetched this on the server and handed the dehydrated cache to a HydrationBoundary.',
      links: []
    },
    {
      title: 'URL State with nuqs',
      description:
        'Pagination, search, and role filters are synced to the URL via nuqs. The useDataTable hook manages the TanStack Table state and debounces filter changes before updating the URL. When the URL changes, React Query automatically refetches because the query key includes the filters.',
      links: [
        {
          title: 'nuqs Documentation',
          url: 'https://nuqs.47ng.com'
        }
      ]
    },
    {
      title: 'Products vs Users Pattern',
      description:
        'Both features now follow the same shape: URL search params → useSuspenseQuery keyed on those filters → DataTable. React Query supplies background refetching, cache sharing across components, and optimistic mutations.',
      links: []
    }
  ]
};
