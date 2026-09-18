import ProfileViewPage from '@/features/profile/components/profile-view-page';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function ProfilePage() {
  useDocumentTitle('Profile');
  return <ProfileViewPage />;
}
