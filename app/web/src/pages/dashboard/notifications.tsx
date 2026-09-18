import NotificationsView from '@/features/notifications/components/notifications-page';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function NotificationsPage() {
  useDocumentTitle('Notifications');
  return <NotificationsView />;
}
