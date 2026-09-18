import PageContainer from '@/components/layout/page-container';
import AdvancedFormPatterns from '@/features/forms/components/advanced-form-patterns';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function AdvancedFormPage() {
  useDocumentTitle('Advanced Form Patterns');
  return (
    <PageContainer
      pageTitle='Advanced Form Patterns'
      pageDescription='Linked fields, async validation, dynamic rows, nested objects, cross-field validation, and form-level errors.'
    >
      <AdvancedFormPatterns />
    </PageContainer>
  );
}
