import { Suspense } from 'react';
import PageContainer from '@/components/layout/page-container';
import { PokemonInfo } from '@/features/react-query-demo/components/pokemon-info';
import { PokemonSkeleton } from '@/features/react-query-demo/components/pokemon-skeleton';
import { reactQueryInfoContent } from '@/features/react-query-demo/info-content';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function ReactQueryPage() {
  useDocumentTitle('React Query');

  return (
    <PageContainer
      pageTitle='React Query'
      // The Next version prefetched on the server and hydrated the cache.
      // An SPA has no server pass, so this is the plain suspense-query half
      // of that pattern.
      pageDescription='Suspense query pattern with React Query.'
      infoContent={reactQueryInfoContent}
    >
      <Suspense fallback={<PokemonSkeleton />}>
        <PokemonInfo />
      </Suspense>
    </PageContainer>
  );
}
