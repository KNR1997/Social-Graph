import * as React from 'react';
import { cn } from '@/lib/utils';

type ImageProps = Omit<React.ComponentProps<'img'>, 'width' | 'height'> & {
  src: string;
  alt: string;
  width?: number | string;
  height?: number | string;
  /** next/image layout prop: stretch to fill the nearest positioned ancestor. */
  fill?: boolean;
  /** Accepted for source compatibility with next/image; ignored. */
  priority?: boolean;
  quality?: number;
  sizes?: string;
  unoptimized?: boolean;
};

/**
 * Drop-in stand-in for next/image. There is no image optimizer in a static
 * SPA build, so this renders a plain <img> and only translates the layout
 * props that affect rendering (`fill`). `priority`/`quality`/`unoptimized`
 * are accepted and dropped so copied call sites compile untouched.
 */
const Image = React.forwardRef<HTMLImageElement, ImageProps>(
  (
    { fill, priority: _priority, quality: _quality, unoptimized: _unoptimized, className, ...props },
    ref
  ) => (
    <img
      ref={ref}
      loading={_priority ? 'eager' : 'lazy'}
      decoding='async'
      className={cn(fill && 'absolute inset-0 h-full w-full object-cover', className)}
      {...props}
    />
  )
);
Image.displayName = 'Image';

// Default-only export, like next/link and next/image, so copied call sites
// keep their `import Image from '...'` form without a lint warning.
export default Image;
