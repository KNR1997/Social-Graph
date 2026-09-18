import { useLocation } from 'react-router-dom';

/** Stand-in for next/navigation's usePathname. */
export function usePathname() {
  return useLocation().pathname;
}
