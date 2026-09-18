import PageContainer from '@/components/layout/page-container';
import FormsShowcasePage from '@/features/forms/components/forms-showcase-page';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function MultiStepFormPage() {
  useDocumentTitle('Multi-Step Form');
  return (
    <PageContainer pageTitle='Multi-Step Form' pageDescription='Multi-step wizard form pattern.'>
      <FormsShowcasePage />
    </PageContainer>
  );
}
