import SignInViewPage from '@/features/auth/components/sign-in-view';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function SignInPage() {
  useDocumentTitle('Sign In');
  return <SignInViewPage />;
}
