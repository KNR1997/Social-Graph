import PageContainer from '@/components/layout/page-container';
import SheetFormDemo from '@/features/forms/components/sheet-form-demo';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function SheetFormPage() {
  useDocumentTitle('Sheet Form');
  return (
    <PageContainer
      pageTitle='Sheet & Dialog Forms'
      pageDescription='Form patterns inside sheets and dialogs with external submit buttons.'
    >
      <SheetFormDemo />
    </PageContainer>
  );
}
