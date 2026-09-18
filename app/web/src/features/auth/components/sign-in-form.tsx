import { useMutation } from '@tanstack/react-query';
import { useLocation, useNavigate } from 'react-router-dom';
import * as z from 'zod';
import { FieldGroup } from '@/components/ui/field';
import Link from '@/components/link';
import { loginMutation } from '@/features/auth/api/mutations';
import { ApiError } from '@/lib/api';
import { useAppForm } from '@/lib/form';

const schema = z.object({
  email: z.string().min(1, 'Email is required').email('Enter a valid email address'),
  // Deliberately no length or complexity rule on sign-in. The policy applies
  // when a password is *chosen*; enforcing it here would lock out anyone whose
  // existing password predates a policy change, and would tell an attacker
  // which guesses are not worth sending.
  password: z.string().min(1, 'Password is required')
});

const AFTER_SIGN_IN = '/dashboard/overview';

export default function SignInForm() {
  const navigate = useNavigate();
  const location = useLocation();
  const { mutateAsync, error, isPending } = useMutation(loginMutation);

  // ProtectedRoute stashes where the visitor was headed before it bounced them
  // here, so a deep link survives the detour through sign-in.
  const from = (location.state as { from?: Location } | null)?.from?.pathname;

  const form = useAppForm({
    defaultValues: { email: '', password: '' },
    validators: { onSubmit: schema },
    onSubmit: async ({ value }) => {
      await mutateAsync(value);
      navigate(from ?? AFTER_SIGN_IN, { replace: true });
    }
  });

  return (
    <form
      className='w-full space-y-4'
      onSubmit={(e) => {
        e.preventDefault();
        form.handleSubmit();
      }}
    >
      <FieldGroup>
        <form.AppField
          name='email'
          children={(field) => (
            <field.TextField
              label='Email'
              type='email'
              autoComplete='email'
              placeholder='you@example.com'
              disabled={isPending}
            />
          )}
        />
        <form.AppField
          name='password'
          children={(field) => (
            <field.TextField
              label='Password'
              type='password'
              autoComplete='current-password'
              placeholder='••••••••'
              disabled={isPending}
            />
          )}
        />
      </FieldGroup>

      <FormError error={error} />

      <form.AppForm>
        <form.SubmitButton className='w-full'>Sign in</form.SubmitButton>
      </form.AppForm>

      <p className='text-muted-foreground text-center text-sm'>
        No account?{' '}
        <Link href='/auth/sign-up' className='text-foreground underline underline-offset-4'>
          Create one
        </Link>
      </p>
    </form>
  );
}

/**
 * Renders the API's rejection.
 *
 * The message is shown as the server worded it rather than being re-derived
 * from the status code, because the server is the side that decided how much
 * to reveal: it answers a wrong password and an unregistered address with the
 * same sentence on purpose, and second-guessing that here would leak the
 * difference the API went out of its way to hide.
 */
function FormError({ error }: { error: Error | null }) {
  if (!error) {
    return null;
  }

  const message =
    error instanceof ApiError ? error.message : 'Something went wrong. Please try again.';

  return (
    <p role='alert' className='text-destructive text-sm'>
      {message}
    </p>
  );
}
