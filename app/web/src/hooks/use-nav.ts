/**
 * Filters navigation by the signed-in user's role.
 *
 * This replaces the Clerk version, which checked organization membership,
 * per-organization permissions, plans and features. Session auth has one axis
 * — the user's role — so the hook is a fraction of the size and, more usefully,
 * no longer has branches that warn at runtime because they need a server call
 * the client cannot make.
 *
 * This is visibility, not access control. Hiding a link stops nobody from
 * typing the URL; the Go API is what refuses the request.
 */

import { useMemo } from 'react';
import { useSession } from '@/hooks/use-session';
import type { NavGroup, NavItem } from '@/types';

/** True when `item` should be visible to a user holding `role`. */
function isVisible(item: NavItem, role: string | undefined): boolean {
  if (!item.access?.role) {
    return true;
  }

  return item.access.role === role;
}

/**
 * Filters a flat list of navigation items, recursing into children.
 *
 * A parent whose children all disappear is dropped too: a collapsible group
 * that opens onto nothing is worse than no group at all.
 */
export function useFilteredNavItems(items: NavItem[]): NavItem[] {
  const { user } = useSession();
  const role = user?.role;

  return useMemo(() => filterItems(items, role), [items, role]);
}

function filterItems(items: NavItem[], role: string | undefined): NavItem[] {
  return items
    .filter((item) => isVisible(item, role))
    .map((item) => {
      if (!item.items?.length) {
        return item;
      }

      return { ...item, items: filterItems(item.items, role) };
    })
    .filter((item) => item.url !== '#' || item.items?.length);
}

/** Filters navigation groups, dropping any left empty. */
export function useFilteredNavGroups(groups: NavGroup[]): NavGroup[] {
  const { user } = useSession();
  const role = user?.role;

  return useMemo(
    () =>
      groups
        .map((group) => ({ ...group, items: filterItems(group.items, role) }))
        .filter((group) => group.items.length > 0),
    [groups, role]
  );
}
