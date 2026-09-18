import { Navigate, Route, Routes } from 'react-router-dom';
import DashboardLayout from '@/layouts/dashboard-layout';
import GuestRoute from './guest-route';
import ProtectedRoute from './protected-route';
import { RouteErrorBoundary } from '@/components/route-error-boundary';

import SignInPage from '@/pages/auth/sign-in';
import SignUpPage from '@/pages/auth/sign-up';
import NotFoundPage from '@/pages/not-found';

import OverviewPage from '@/pages/dashboard/overview';
import ProductsPage from '@/pages/dashboard/products';
import ProductViewRoute from '@/pages/dashboard/product-view';
import UsersPage from '@/pages/dashboard/users';
import ReactQueryPage from '@/pages/dashboard/react-query';
import KanbanPage from '@/pages/dashboard/kanban';
import ChatPage from '@/pages/dashboard/chat';
import AiChatPage from '@/pages/dashboard/ai-chat';
import NotificationsPage from '@/pages/dashboard/notifications';
import IconsPage from '@/pages/dashboard/icons';
import ProfilePage from '@/pages/dashboard/profile';
import BasicFormPage from '@/pages/dashboard/forms/basic';
import AdvancedFormPage from '@/pages/dashboard/forms/advanced';
import MultiStepFormPage from '@/pages/dashboard/forms/multi-step';
import SheetFormPage from '@/pages/dashboard/forms/sheet-form';

/**
 * The route tree.
 *
 * Mapping notes (from the Next version this was ported from):
 * - `redirect()` in a server page  -> <Navigate replace />
 * - middleware (src/proxy.ts)      -> <ProtectedRoute /> layout route
 * - error.tsx / not-found.tsx      -> <RouteErrorBoundary />
 *
 * The auth pages sit under <GuestRoute /> and everything else under
 * <ProtectedRoute />, so the two states are visible in the tree rather than
 * decided inside each page. Neither is a security boundary: the Go API is
 * where access is enforced.
 *
 * The Clerk-only routes (/dashboard/workspaces, /workspaces/team, /billing and
 * /exclusive) were removed along with Clerk. They were thin wrappers over its
 * organization and billing widgets, and there is no local equivalent to point
 * them at.
 */
export function AppRoutes() {
  return (
    <Routes>
      <Route element={<GuestRoute />}>
        <Route path='/auth' element={<Navigate to='/auth/sign-in' replace />} />
        <Route path='/auth/sign-in' element={<SignInPage />} />
        <Route path='/auth/sign-up' element={<SignUpPage />} />
      </Route>

      <Route element={<ProtectedRoute />}>
        <Route path='/' element={<Navigate to='/dashboard/overview' replace />} />

        <Route
          path='/dashboard'
          element={
            <RouteErrorBoundary>
              <DashboardLayout />
            </RouteErrorBoundary>
          }
        >
          <Route index element={<Navigate to='/dashboard/overview' replace />} />
          <Route path='overview' element={<OverviewPage />} />

          <Route path='product' element={<ProductsPage />} />
          <Route path='product/:productId' element={<ProductViewRoute />} />
          <Route path='users' element={<UsersPage />} />
          <Route path='react-query' element={<ReactQueryPage />} />

          <Route path='forms' element={<Navigate to='/dashboard/forms/basic' replace />} />
          <Route path='forms/basic' element={<BasicFormPage />} />
          <Route path='forms/advanced' element={<AdvancedFormPage />} />
          <Route path='forms/multi-step' element={<MultiStepFormPage />} />
          <Route path='forms/sheet-form' element={<SheetFormPage />} />

          <Route path='kanban' element={<KanbanPage />} />
          <Route path='chat' element={<ChatPage />} />
          <Route path='ai-chat' element={<AiChatPage />} />
          <Route path='notifications' element={<NotificationsPage />} />
          <Route path='elements/icons' element={<IconsPage />} />

          <Route path='profile' element={<ProfilePage />} />

          <Route path='*' element={<NotFoundPage />} />
        </Route>
      </Route>

      <Route path='*' element={<NotFoundPage />} />
    </Routes>
  );
}
