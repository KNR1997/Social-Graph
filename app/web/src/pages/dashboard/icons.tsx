import IconsViewPage from '@/features/elements/components/icons-view-page';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function IconsPage() {
  useDocumentTitle('Icons');
  return <IconsViewPage />;
}
