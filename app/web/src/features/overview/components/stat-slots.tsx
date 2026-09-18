import { Suspense } from 'react';
import { useSuspenseQuery } from '@tanstack/react-query';
import { ErrorBoundary, type FallbackProps } from 'react-error-boundary';
import { useTransition } from 'react';
import { delay } from '@/constants/mock-api';
import { Button } from '@/components/ui/button';
import { Icons } from '@/components/icons';
import { StatsErrorAlert } from './stats-error';
import { AreaGraph } from './area-graph';
import { BarGraph } from './bar-graph';
import { PieGraph } from './pie-graph';
import { RecentSales } from './recent-sales';
import { AreaGraphSkeleton } from './area-graph-skeleton';
import { BarGraphSkeleton } from './bar-graph-skeleton';
import { PieGraphSkeleton } from './pie-graph-skeleton';
import { RecentSalesSkeleton } from './recent-sales-skeleton';

// ============================================================
// Overview stat slots
// ============================================================
// The Next version modelled these as four parallel routes
// (@area_stats, @bar_stats, @pie_stats, @sales), each an async server
// component that awaited delay() so the four cards streamed in one at a time.
//
// There is no streaming SSR here, so the same staggered reveal is reproduced
// with a suspending query per slot: useSuspenseQuery hits the Suspense
// boundary, the skeleton (the old loading.tsx) shows, and the chart swaps in
// when its delay resolves.
// ============================================================

function useSlotDelay(slot: string, ms: number) {
  useSuspenseQuery({
    queryKey: ['overview', slot],
    queryFn: () => delay(ms).then(() => true),
    staleTime: Infinity
  });
}

function AreaStats() {
  useSlotDelay('area_stats', 2000);
  return <AreaGraph />;
}

function BarStats() {
  useSlotDelay('bar_stats', 1000);
  return <BarGraph />;
}

function PieStats() {
  useSlotDelay('pie_stats', 1000);
  return <PieGraph />;
}

function Sales() {
  useSlotDelay('sales', 3000);
  return <RecentSales />;
}

/** Port of the shared overview error.tsx. */
function StatsErrorFallback({ error, resetErrorBoundary }: FallbackProps) {
  const [isPending, startTransition] = useTransition();
  // react-error-boundary v6 types `error` as unknown.
  const message = error instanceof Error ? error.message : String(error);

  const retry = () => {
    startTransition(() => {
      resetErrorBoundary();
    });
  };

  return (
    <StatsErrorAlert
      message={`Failed to load statistics: ${message}`}
      action={
        <>
          <Button variant='outline' size='sm' onClick={retry} disabled={isPending}>
            {isPending ? (
              <>
                <Icons.spinner className='mr-2 h-4 w-4 animate-spin' aria-hidden='true' />
                Retrying...
              </>
            ) : (
              'Try again'
            )}
          </Button>
          <span role='status' aria-live='polite' className='sr-only'>
            {isPending ? 'Retrying' : ''}
          </span>
        </>
      }
    />
  );
}

function Slot({
  children,
  fallback
}: {
  children: React.ReactNode;
  fallback: React.ReactNode;
}) {
  return (
    <ErrorBoundary FallbackComponent={StatsErrorFallback}>
      <Suspense fallback={fallback}>{children}</Suspense>
    </ErrorBoundary>
  );
}

export const AreaStatsSlot = () => (
  <Slot fallback={<AreaGraphSkeleton />}>
    <AreaStats />
  </Slot>
);

export const BarStatsSlot = () => (
  <Slot fallback={<BarGraphSkeleton />}>
    <BarStats />
  </Slot>
);

export const PieStatsSlot = () => (
  <Slot fallback={<PieGraphSkeleton />}>
    <PieStats />
  </Slot>
);

export const SalesSlot = () => (
  <Slot fallback={<RecentSalesSkeleton />}>
    <Sales />
  </Slot>
);
