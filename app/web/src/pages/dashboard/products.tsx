import { Suspense } from 'react';
import { cn } from '@/lib/utils';
// config
import { productInfoContent } from '@/config/infoconfig';
// hooks
import { useDocumentTitle } from '@/hooks/use-document-title';
// features
import { ProductTable } from '@/features/products/components/product-tables';
// components
import Link from '@/components/link';
import { Icons } from '@/components/icons';
import { buttonVariants } from '@/components/ui/button';
import PageContainer from '@/components/layout/page-container';
import { DataTableSkeleton } from '@/components/ui/table/data-table-skeleton';

export default function ProductsPage() {
  useDocumentTitle('Products');

  return (
    <PageContainer
      pageTitle='Products'
      pageDescription='Manage products (React Query + nuqs table pattern.)'
      infoContent={productInfoContent}
      pageHeaderAction={
        <Link href='/dashboard/product/new' className={cn(buttonVariants(), 'text-xs md:text-sm')}>
          <Icons.add className='mr-2 h-4 w-4' /> Add New
        </Link>
      }
    >
      {/* The Next version wrapped ProductTable in a server component that
          read searchParamsCache and prefetched into a HydrationBoundary.
          ProductTable reads the same params itself via nuqs, so the Suspense
          boundary (formerly product/loading.tsx) is all that is needed. */}
      <Suspense fallback={<DataTableSkeleton columnCount={7} rowCount={10} filterCount={2} />}>
        <ProductTable />
      </Suspense>
    </PageContainer>
  );
}
