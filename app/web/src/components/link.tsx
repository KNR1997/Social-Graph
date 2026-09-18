import { Link as RouterLink, type LinkProps as RouterLinkProps } from 'react-router-dom';
import * as React from 'react';

type LinkProps = Omit<RouterLinkProps, 'to'> & {
  href: string;
  /** Accepted for source compatibility with next/link; ignored. */
  prefetch?: boolean | null;
  scroll?: boolean;
};

/**
 * Drop-in stand-in for next/link so copied components keep using `href`.
 * External/anchor targets fall through to a plain <a>, matching next/link.
 */
const Link = React.forwardRef<HTMLAnchorElement, LinkProps>(
  ({ href, prefetch: _prefetch, scroll: _scroll, replace, children, ...props }, ref) => {
    const isExternal = /^([a-z]+:)?\/\//i.test(href) || href.startsWith('mailto:') ||
      href.startsWith('tel:') || href.startsWith('#');

    if (isExternal) {
      return (
        <a ref={ref} href={href} {...props}>
          {children}
        </a>
      );
    }

    return (
      <RouterLink ref={ref} to={href} replace={replace} {...props}>
        {children}
      </RouterLink>
    );
  }
);
Link.displayName = 'Link';

// Default-only export, like next/link and next/image, so copied call sites
// keep their `import Link from '...'` form without a lint warning.
export default Link;
