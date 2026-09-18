import PageContainer from '@/components/layout/page-container';
import DemoForm from '@/components/forms/demo-form';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function BasicFormPage() {
  useDocumentTitle('Basic Form');
  return (
    <PageContainer
      pageTitle='Basic Form'
      pageDescription='A comprehensive form demo with all field types.'
    >
      <DemoForm />
    </PageContainer>
  );
}
