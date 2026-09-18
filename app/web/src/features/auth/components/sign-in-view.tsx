import AuthLayout from './auth-layout';
import SignInForm from './sign-in-form';

export default function SignInViewPage() {
  return (
    <AuthLayout title='Welcome back' subtitle='Sign in to continue to your dashboard.'>
      <SignInForm />
    </AuthLayout>
  );
}
