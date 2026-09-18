import { NavGroup } from '@/types';

/**
 * Navigation configuration with role-based visibility.
 *
 * This drives both the sidebar and the Cmd+K bar, so an item added here shows
 * up in both.
 *
 * Access control:
 * Each item may carry an `access` object. It has one field, `role`, checked
 * against the signed-in user's role from GET /v1/auth/me:
 *
 *   access: { role: 'admin' }
 *
 * An item with no `access` is visible to everyone signed in. Filtering happens
 * in `useFilteredNavGroups` (src/hooks/use-nav.ts).
 *
 * This is visibility only. Hiding a link does not protect the page behind it,
 * let alone the data: the Go API enforces access on every request, and that is
 * the check that counts.
 *
 * The Clerk version of this file also supported `permission`, `plan`,
 * `feature` and `requireOrg`. Those described Clerk organizations and billing,
 * neither of which exists here, and the plan/feature branches never worked
 * client-side anyway -- they logged a warning and showed the item.
 */
export const navGroups: NavGroup[] = [
  {
    label: 'Overview',
    items: [
      {
        title: 'Dashboard',
        url: '/dashboard/overview',
        icon: 'dashboard',
        isActive: false,
        shortcut: ['d', 'd'],
        items: []
      }
      // {
      //   title: 'Product',
      //   url: '/dashboard/product',
      //   icon: 'product',
      //   shortcut: ['p', 'p'],
      //   isActive: false,
      //   items: []
      // },
      // {
      //   title: 'Users',
      //   url: '/dashboard/users',
      //   icon: 'teams',
      //   shortcut: ['u', 'u'],
      //   isActive: false,
      //   items: [],
      //   access: { role: 'admin' }
      // },
      // {
      //   title: 'Kanban',
      //   url: '/dashboard/kanban',
      //   icon: 'kanban',
      //   shortcut: ['k', 'k'],
      //   isActive: false,
      //   items: []
      // },
      // {
      //   title: 'Chat',
      //   url: '/dashboard/chat',
      //   icon: 'chat',
      //   shortcut: ['c', 'c'],
      //   isActive: false,
      //   items: []
      // },
      // {
      //   title: 'AI Chat',
      //   url: '/dashboard/ai-chat',
      //   icon: 'sparkles',
      //   shortcut: ['a', 'i'],
      //   isActive: false,
      //   items: []
      // }
    ]
  },
  // {
  //   label: 'Elements',
  //   items: [
  //     {
  //       title: 'Forms',
  //       url: '#',
  //       icon: 'forms',
  //       isActive: true,
  //       items: [
  //         {
  //           title: 'Basic Form',
  //           url: '/dashboard/forms/basic',
  //           icon: 'forms',
  //           shortcut: ['f', 'f']
  //         },
  //         {
  //           title: 'Multi-Step Form',
  //           url: '/dashboard/forms/multi-step',
  //           icon: 'forms'
  //         },
  //         {
  //           title: 'Sheet & Dialog',
  //           url: '/dashboard/forms/sheet-form',
  //           icon: 'forms'
  //         },
  //         {
  //           title: 'Advanced Patterns',
  //           url: '/dashboard/forms/advanced',
  //           icon: 'forms'
  //         }
  //       ]
  //     },
  //     {
  //       title: 'React Query',
  //       url: '/dashboard/react-query',
  //       icon: 'code',
  //       isActive: false,
  //       items: []
  //     },
  //     {
  //       title: 'Icons',
  //       url: '/dashboard/elements/icons',
  //       icon: 'palette',
  //       isActive: false,
  //       items: []
  //     }
  //   ]
  // },
  {
    label: 'Account',
    items: [
      {
        title: 'Profile',
        url: '/dashboard/profile',
        icon: 'profile',
        shortcut: ['m', 'm'],
        isActive: false,
        items: []
      },
      {
        title: 'Notifications',
        url: '/dashboard/notifications',
        icon: 'notification',
        shortcut: ['n', 'n'],
        isActive: false,
        items: []
      }
    ]
  }
];
