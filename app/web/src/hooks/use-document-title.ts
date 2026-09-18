import * as React from 'react';

const SUFFIX = 'Shadcn Dashboard';

/**
 * Replaces the per-page `metadata` exports of the Next version. Static OG and
 * twitter tags live in index.html; only the title is dynamic per route.
 */
export function useDocumentTitle(title?: string) {
  React.useEffect(() => {
    const previous = document.title;
    document.title = title ? `${title} | ${SUFFIX}` : SUFFIX;
    return () => {
      document.title = previous;
    };
  }, [title]);
}
