import { Suspense } from 'react';
import PageContainer from '@/components/layout/page-container';
import { DataTableSkeleton } from '@/components/ui/table/data-table-skeleton';
import { UsersTable } from '@/features/users/components/users-table';
import { usersInfoContent } from '@/features/users/info-content';
import { UserFormSheetTrigger } from '@/features/users/components/user-form-sheet';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function UsersPage() {
  useDocumentTitle('Users');

  return (
    <PageContainer
      pageTitle='Users'
      pageDescription='Manage users (React Query + nuqs table pattern.)'
      infoContent={usersInfoContent}
      pageHeaderAction={<UserFormSheetTrigger />}
    >
      <Suspense fallback={<DataTableSkeleton columnCount={6} rowCount={10} filterCount={2} />}>
        <UsersTable />
      </Suspense>
    </PageContainer>
  );
}
