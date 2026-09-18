/**
 * Tiny cookie helpers. These replace the two `cookies()` reads from
 * next/headers (`active_theme` in the root layout, `sidebar_state` in the
 * dashboard layout). Cookies rather than localStorage are kept deliberately:
 * the existing ActiveThemeProvider already writes `active_theme` as a cookie,
 * and the pre-paint script in index.html reads it synchronously.
 */
export function getCookie(name: string): string | undefined {
  if (typeof document === 'undefined') return undefined;
  const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : undefined;
}

export function setCookie(name: string, value: string, maxAgeSeconds = 60 * 60 * 24 * 365) {
  document.cookie = `${name}=${encodeURIComponent(value)}; path=/; max-age=${maxAgeSeconds}; SameSite=Lax`;
}
