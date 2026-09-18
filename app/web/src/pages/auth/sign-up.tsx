import SignUpViewPage from '@/features/auth/components/sign-up-view';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function SignUpPage() {
  useDocumentTitle('Sign Up');
  return <SignUpViewPage />;
}
