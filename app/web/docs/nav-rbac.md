# Navigation access control

How the sidebar and the Cmd+K bar decide what to show.

**Navigation visibility is UX, not security.** Hiding a link stops nobody: a
visitor can type the URL, edit the bundle, or call the API directly. Access is
enforced in the Go service, where every route outside `/v1/auth/{register,login}`
sits behind `RequireSession`, and that is the check that counts. Everything
below is about not showing people doors that will not open for them.

## Core files

| File | Role |
| --- | --- |
| `src/config/nav-config.ts` | The navigation tree, with an optional `access` per item |
| `src/hooks/use-nav.ts` | `useFilteredNavItems` / `useFilteredNavGroups` |
| `src/hooks/use-session.ts` | `useSession`, the signed-in user and their role |
| `src/types/index.ts` | `NavItem`, `PermissionCheck` |

## The model

One axis: the user's role, as returned by `GET /v1/auth/me`.

```ts
export type UserRole = 'member' | 'admin';

export interface PermissionCheck {
  role?: UserRole;
}
```

An item with no `access` is visible to anyone signed in. An item with
`access: { role: 'admin' }` is visible only to admins:

```ts
{
  title: 'Users',
  url: '/dashboard/users',
  icon: 'teams',
  access: { role: 'admin' }
}
```

Filtering recurses into `items`, and a parent whose children all disappear is
dropped along with them, so a collapsible group never opens onto nothing.
A group left with no items is removed too.

## Usage

```tsx
import { navGroups } from '@/config/nav-config';
import { useFilteredNavGroups } from '@/hooks/use-nav';

const groups = useFilteredNavGroups(navGroups);
```

The filter is synchronous and reads one React Query cache entry, so there is no
loading state and no flash of items that then vanish. On the very first load,
before `GET /v1/auth/me` resolves, the role is `undefined` and role-gated items
are hidden — the conservative direction, and moot in practice because
`ProtectedRoute` renders a spinner until the session is known.

To gate rendering rather than navigation, read the session directly:

```tsx
const { user } = useSession();
if (user?.role !== 'admin') return null;
```

Or `useHasRole('admin')` for the common case.

## Adding a role

1. Add it to `entity.Role` and `ParseRole` in
   `internal/domain/auth/entity/user.go`.
2. Extend the `users_role_known` CHECK constraint in a new migration.
3. Add it to `UserRole` in `src/features/auth/api/types.ts`.

Steps 1 and 3 are a duplicated list, and the duplication is deliberate — the
alternative is generating the client types from the Go source, which is not
worth it for two values. If they drift, an unknown role simply hides
role-gated items rather than failing loudly, so keep them in step.

## What changed from the Clerk version

The previous system checked organization membership, per-organization
permissions, subscription plans, and features. All four came from Clerk and
went with it.

The plan and feature checks were never really working: they needed Clerk's
server-side `has()`, so the client-side hook logged a warning and showed the
item anyway. If you need finer-grained authorization than a role, add it to the
Go domain first and expose it on `/v1/auth/me` — a permission the client can
name but the server cannot enforce is worse than no permission at all.
