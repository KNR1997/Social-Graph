import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import type { SessionUser } from '@/features/auth/api/types';

interface UserAvatarProfileProps {
  className?: string;
  showInfo?: boolean;
  user: Pick<SessionUser, 'name' | 'email'> | null;
}

/**
 * The signed-in user's avatar.
 *
 * There is no image: Clerk hosted an avatar per user and our API stores no
 * such thing, so this renders initials. Adding real avatars means an upload
 * endpoint and somewhere to put the file, which is a feature rather than a
 * detail of authentication.
 */
export function UserAvatarProfile({ className, showInfo = false, user }: UserAvatarProfileProps) {
  return (
    <div className='flex items-center gap-2'>
      <Avatar className={className}>
        <AvatarFallback className='rounded-lg'>{initials(user?.name)}</AvatarFallback>
      </Avatar>

      {showInfo && (
        <div className='grid flex-1 text-left text-sm leading-tight'>
          <span className='truncate font-semibold'>{user?.name ?? ''}</span>
          <span className='truncate text-xs'>{user?.email ?? ''}</span>
        </div>
      )}
    </div>
  );
}

/**
 * Builds up to two initials from a display name.
 *
 * Iterating with the spread operator rather than indexing keeps this correct
 * for names outside the basic multilingual plane, where a single character is
 * two UTF-16 code units and `name[0]` would slice one in half.
 */
function initials(name: string | undefined): string {
  if (!name) {
    return 'CN';
  }

  const parts = name.trim().split(/\s+/).filter(Boolean);
  const letters = parts.slice(0, 2).map((part) => [...part][0] ?? '');

  return letters.join('').toUpperCase() || 'CN';
}
