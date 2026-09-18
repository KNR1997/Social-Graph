import { ErrorBoundary, type FallbackProps } from 'react-error-boundary';
import * as React from 'react';
import { useLocation } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { isNotFoundError } from '@/lib/not-found';
import NotFoundPage from '@/pages/not-found';

function DefaultFallback({ error, resetErrorBoundary }: FallbackProps) {
  // notFound() throws through here, mirroring how Next unwound a
  // notFound() call to the nearest not-found boundary.
  if (isNotFoundError(error)) return <NotFoundPage />;

  // react-error-boundary v6 types `error` as unknown.
  const message = error instanceof Error ? error.message : String(error);

  return (
    <div className='flex flex-1 flex-col items-center justify-center gap-4 p-8 text-center'>
      <h2 className='text-xl font-semibold'>Something went wrong</h2>
      <p className='text-muted-foreground max-w-md text-sm'>{message}</p>
      <Button onClick={resetErrorBoundary} variant='outline' size='sm'>
        Try again
      </Button>
    </div>
  );
}

/**
 * Stands in for the app-router `error.tsx` convention. Keying the boundary on
 * the pathname resets it on navigation, which Next did implicitly.
 */
export function RouteErrorBoundary({
  children,
  fallback
}: {
  children: React.ReactNode;
  fallback?: React.ComponentType<FallbackProps>;
}) {
  const { pathname } = useLocation();
  return (
    <ErrorBoundary key={pathname} FallbackComponent={fallback ?? DefaultFallback}>
      {children}
    </ErrorBoundary>
  );
}
