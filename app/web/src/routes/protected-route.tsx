import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { Icons } from '@/components/icons';
import { useSession } from '@/hooks/use-session';

/**
 * The authentication guard, replacing Clerk's SignedIn/RedirectToSignIn pair
 * and, before that, the Next version's clerkMiddleware in src/proxy.ts.
 *
 * This is a UX affordance, not a security boundary. It decides what to render;
 * it does not decide what data anyone can reach. Every protected route in the
 * Go API is behind RequireSession there, which is where access is actually
 * enforced — a determined visitor can edit this component out of the bundle
 * and still get a 401 from every request it would have hidden.
 *
 * The three-way branch is the important part. Rendering the redirect while the
 * session request is still in flight would bounce every signed-in user to the
 * login page on a hard refresh, so "not known yet" gets its own arm.
 */
export default function ProtectedRoute() {
  const { isPending, isSignedIn } = useSession();
  const location = useLocation();

  if (isPending) {
    return (
      <div className='flex min-h-screen items-center justify-center'>
        <Icons.spinner className='text-muted-foreground size-6 animate-spin' aria-label='Loading' />
      </div>
    );
  }

  if (!isSignedIn) {
    // Carry where they were headed, so signing in returns them there instead of
    // dumping everyone on the dashboard root.
    return <Navigate to='/auth/sign-in' replace state={{ from: location }} />;
  }

  return <Outlet />;
}
