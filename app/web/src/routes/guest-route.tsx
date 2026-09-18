import { Navigate, Outlet } from 'react-router-dom';
import { Icons } from '@/components/icons';
import { useSession } from '@/hooks/use-session';

/**
 * The inverse of ProtectedRoute: keeps a signed-in user off the sign-in and
 * sign-up pages.
 *
 * Clerk did this implicitly, by having its own components redirect once they
 * saw a session. With our own forms it has to be a route, or a signed-in user
 * following a bookmark to /auth/sign-in gets a login form for an account they
 * are already using.
 *
 * It waits out the pending state for the same reason ProtectedRoute does:
 * rendering the form before the session request lands would flash a login page
 * at somebody who is already signed in.
 */
export default function GuestRoute() {
  const { isPending, isSignedIn } = useSession();

  if (isPending) {
    return (
      <div className='flex min-h-screen items-center justify-center'>
        <Icons.spinner className='text-muted-foreground size-6 animate-spin' aria-label='Loading' />
      </div>
    );
  }

  if (isSignedIn) {
    return <Navigate to='/dashboard/overview' replace />;
  }

  return <Outlet />;
}
