import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import * as z from 'zod';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { FieldGroup } from '@/components/ui/field';
import { Badge } from '@/components/ui/badge';
import { changePasswordMutation, updateProfileMutation } from '@/features/auth/api/mutations';
import { useSignedInUser } from '@/hooks/use-session';
import { ApiError } from '@/lib/api';
import { useAppForm } from '@/lib/form';

/**
 * The profile page, replacing Clerk's <UserProfile /> widget.
 *
 * It covers what the session API actually supports — display name and
 * password — rather than reproducing Clerk's full account surface. Email is
 * shown read-only because it is the account identity: changing it needs a
 * verification flow, and an endpoint that swapped it without one would be a
 * way to take over an account from a borrowed session.
 */
export default function ProfileViewPage() {
  const user = useSignedInUser();

  return (
    <div className='flex w-full flex-col gap-4 p-4'>
      <Card>
        <CardHeader>
          <CardTitle>Profile</CardTitle>
          <CardDescription>Your account details.</CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='flex items-center justify-between gap-4'>
            <div className='space-y-1'>
              <p className='text-muted-foreground text-sm'>Email</p>
              <p className='text-sm font-medium'>{user.email}</p>
            </div>
            <Badge variant='secondary'>{user.role}</Badge>
          </div>
          <NameForm currentName={user.name} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Password</CardTitle>
          <CardDescription>Changing your password signs out every other device.</CardDescription>
        </CardHeader>
        <CardContent>
          <PasswordForm />
        </CardContent>
      </Card>
    </div>
  );
}

const nameSchema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(200, 'Name is too long')
});

function NameForm({ currentName }: { currentName: string }) {
  const { mutateAsync, isPending } = useMutation(updateProfileMutation);

  const form = useAppForm({
    defaultValues: { name: currentName },
    validators: { onSubmit: nameSchema },
    onSubmit: async ({ value }) => {
      try {
        await mutateAsync(value.name);
        toast.success('Profile updated');
      } catch (error) {
        toast.error(messageFor(error));
      }
    }
  });

  return (
    <form
      className='space-y-4'
      onSubmit={(e) => {
        e.preventDefault();
        form.handleSubmit();
      }}
    >
      <FieldGroup>
        <form.AppField
          name='name'
          children={(field) => (
            <field.TextField label='Name' autoComplete='name' disabled={isPending} />
          )}
        />
      </FieldGroup>
      <form.AppForm>
        <form.SubmitButton>Save</form.SubmitButton>
      </form.AppForm>
    </form>
  );
}

const MIN_PASSWORD_LENGTH = 8;

const passwordSchema = z
  .object({
    current_password: z.string().min(1, 'Enter your current password'),
    new_password: z
      .string()
      .min(MIN_PASSWORD_LENGTH, `Password must be at least ${MIN_PASSWORD_LENGTH} characters`)
      .max(128, 'Password must be at most 128 characters'),
    confirmPassword: z.string().min(1, 'Confirm your new password')
  })
  .refine((values) => values.new_password === values.confirmPassword, {
    message: 'Passwords do not match',
    path: ['confirmPassword']
  });

function PasswordForm() {
  const { mutateAsync, isPending } = useMutation(changePasswordMutation);

  const form = useAppForm({
    defaultValues: { current_password: '', new_password: '', confirmPassword: '' },
    validators: { onSubmit: passwordSchema },
    onSubmit: async ({ value, formApi }) => {
      try {
        await mutateAsync({
          current_password: value.current_password,
          new_password: value.new_password
        });
        // The API re-issued this browser's cookie, so we stay signed in.
        toast.success('Password changed. Other devices have been signed out.');
        formApi.reset();
      } catch (error) {
        toast.error(messageFor(error));
      }
    }
  });

  return (
    <form
      className='space-y-4'
      onSubmit={(e) => {
        e.preventDefault();
        form.handleSubmit();
      }}
    >
      <FieldGroup>
        <form.AppField
          name='current_password'
          children={(field) => (
            <field.TextField
              label='Current password'
              type='password'
              autoComplete='current-password'
              disabled={isPending}
            />
          )}
        />
        <form.AppField
          name='new_password'
          children={(field) => (
            <field.TextField
              label='New password'
              type='password'
              autoComplete='new-password'
              description={`At least ${MIN_PASSWORD_LENGTH} characters.`}
              disabled={isPending}
            />
          )}
        />
        <form.AppField
          name='confirmPassword'
          children={(field) => (
            <field.TextField
              label='Confirm new password'
              type='password'
              autoComplete='new-password'
              disabled={isPending}
            />
          )}
        />
      </FieldGroup>
      <form.AppForm>
        <form.SubmitButton>Change password</form.SubmitButton>
      </form.AppForm>
    </form>
  );
}

/** The API's message when it gave one, a generic fallback otherwise. */
function messageFor(error: unknown): string {
  return error instanceof ApiError ? error.message : 'Something went wrong. Please try again.';
}
