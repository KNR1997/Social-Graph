import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from '@/components/ui/sidebar';
import { Icons } from '@/components/icons';

/**
 * The sidebar header, replacing <OrgSwitcher />.
 *
 * The switcher listed the user's Clerk organizations and set the active one.
 * Session auth has no notion of an organization, so there is nothing to switch
 * between and this is a static brand block instead. If multi-tenancy arrives
 * later, this is where the switcher goes back.
 */
export function WorkspaceBrand() {
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton size='lg' className='cursor-default hover:bg-transparent'>
          <div className='bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg'>
            <Icons.logo className='size-4' aria-hidden />
          </div>
          <div className='grid flex-1 text-left text-sm leading-tight'>
            <span className='truncate font-semibold'>Acme Inc</span>
            <span className='text-muted-foreground truncate text-xs'>Dashboard</span>
          </div>
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
