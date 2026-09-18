import KanbanViewPage from '@/features/kanban/components/kanban-view-page';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function KanbanPage() {
  useDocumentTitle('Kanban view');
  return <KanbanViewPage />;
}
