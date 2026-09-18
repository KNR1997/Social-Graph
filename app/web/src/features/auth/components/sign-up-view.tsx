import AuthLayout from './auth-layout';
import SignUpForm from './sign-up-form';

export default function SignUpViewPage() {
  return (
    <AuthLayout title='Create an account' subtitle='Enter your details to get started.'>
      <SignUpForm />
    </AuthLayout>
  );
}
