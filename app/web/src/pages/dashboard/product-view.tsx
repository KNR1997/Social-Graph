import { Suspense } from 'react';
import { useParams } from 'react-router-dom';
import PageContainer from '@/components/layout/page-container';
import FormCardSkeleton from '@/components/form-card-skeleton';
import ProductViewPage from '@/features/products/components/product-view-page';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function ProductViewRoute() {
  const { productId = 'new' } = useParams();
  useDocumentTitle(productId === 'new' ? 'New product' : 'Product view');

  return (
    <PageContainer>
      <div className='flex-1 space-y-4'>
        <Suspense fallback={<FormCardSkeleton />}>
          {/* keyed so switching product ids remounts and re-suspends */}
          <ProductViewPage key={productId} productId={productId} />
        </Suspense>
      </div>
    </PageContainer>
  );
}
