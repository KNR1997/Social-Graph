import { useMutation } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import * as z from 'zod';
import { FieldGroup } from '@/components/ui/field';
import Link from '@/components/link';
import { registerMutation } from '@/features/auth/api/mutations';
import { ApiError } from '@/lib/api';
import { useAppForm } from '@/lib/form';

/**
 * Mirrors the password policy in the Go domain (entity.ValidatePassword).
 *
 * This is a duplicate of a server-side rule, and it stays a duplicate on
 * purpose: the client copy exists to give immediate feedback, and the server
 * copy is the one that decides. If they drift, the server wins and the form
 * shows its rejection.
 */
const MIN_PASSWORD_LENGTH = 8;
const MAX_PASSWORD_LENGTH = 128;

const schema = z
  .object({
    name: z.string().trim().min(1, 'Name is required').max(200, 'Name is too long'),
    email: z.string().min(1, 'Email is required').email('Enter a valid email address'),
    password: z
      .string()
      .min(MIN_PASSWORD_LENGTH, `Password must be at least ${MIN_PASSWORD_LENGTH} characters`)
      .max(MAX_PASSWORD_LENGTH, `Password must be at most ${MAX_PASSWORD_LENGTH} characters`),
    confirmPassword: z.string().min(1, 'Confirm your password')
  })
  .refine((values) => values.password === values.confirmPassword, {
    message: 'Passwords do not match',
    path: ['confirmPassword']
  });

const AFTER_SIGN_UP = '/dashboard/overview';

export default function SignUpForm() {
  const navigate = useNavigate();
  const { mutateAsync, error, isPending } = useMutation(registerMutation);

  const form = useAppForm({
    defaultValues: { name: '', email: '', password: '', confirmPassword: '' },
    validators: { onSubmit: schema },
    onSubmit: async ({ value }) => {
      // confirmPassword exists only to catch typos; it is never sent.
      await mutateAsync({ name: value.name, email: value.email, password: value.password });
      navigate(AFTER_SIGN_UP, { replace: true });
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
          name='name'
          children={(field) => (
            <field.TextField
              label='Name'
              autoComplete='name'
              placeholder='Ada Lovelace'
              disabled={isPending}
            />
          )}
        />
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
              autoComplete='new-password'
              placeholder='••••••••'
              description={`At least ${MIN_PASSWORD_LENGTH} characters.`}
              disabled={isPending}
            />
          )}
        />
        <form.AppField
          name='confirmPassword'
          children={(field) => (
            <field.TextField
              label='Confirm password'
              type='password'
              autoComplete='new-password'
              placeholder='••••••••'
              disabled={isPending}
            />
          )}
        />
      </FieldGroup>

      {error && (
        <p role='alert' className='text-destructive text-sm'>
          {error instanceof ApiError ? error.message : 'Something went wrong. Please try again.'}
        </p>
      )}

      <form.AppForm>
        <form.SubmitButton className='w-full'>Create account</form.SubmitButton>
      </form.AppForm>

      <p className='text-muted-foreground text-center text-sm'>
        Already have an account?{' '}
        <Link href='/auth/sign-in' className='text-foreground underline underline-offset-4'>
          Sign in
        </Link>
      </p>
    </form>
  );
}
