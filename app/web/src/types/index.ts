import { Icons } from '@/components/icons';
import type { UserRole } from '@/features/auth/api/types';

/**
 * Who may see a navigation item.
 *
 * Reduced to a single role when Clerk was removed: organizations, plans and
 * per-organization permissions went with it, and inventing fields the session
 * cannot answer would only reintroduce the checks that used to warn at runtime
 * and then show the item anyway.
 */
export interface PermissionCheck {
  role?: UserRole;
}

export interface NavItem {
  title: string;
  url: string;
  disabled?: boolean;
  external?: boolean;
  shortcut?: [string, string];
  icon?: keyof typeof Icons;
  label?: string;
  description?: string;
  isActive?: boolean;
  items?: NavItem[];
  access?: PermissionCheck;
}

export interface NavGroup {
  label: string;
  items: NavItem[];
}

export interface NavItemWithChildren extends NavItem {
  items: NavItemWithChildren[];
}

export interface NavItemWithOptionalChildren extends NavItem {
  items?: NavItemWithChildren[];
}

export interface FooterItem {
  title: string;
  items: {
    title: string;
    href: string;
    external?: boolean;
  }[];
}

export type MainNavItem = NavItemWithOptionalChildren;

export type SidebarNavItem = NavItemWithChildren;
