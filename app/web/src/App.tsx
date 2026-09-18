import { BrowserRouter } from 'react-router-dom';
import { NuqsAdapter } from 'nuqs/adapters/react-router/v7';
import Providers from '@/components/layout/providers';
import { Toaster } from '@/components/ui/sonner';
import ThemeProvider from '@/components/themes/theme-provider';
import { ActiveThemeProvider } from '@/components/themes/active-theme';
import { DEFAULT_THEME, THEMES } from '@/components/themes/theme.config';
import { getCookie } from '@/lib/cookies';
import { AppRoutes } from '@/routes';

// Replaces the `cookies()` read in the Next root layout. index.html already
// stamped data-theme before first paint; this hands the same value to the
// provider so React state and the DOM agree from the first render.
function readActiveTheme() {
  const value = getCookie('active_theme');
  return THEMES.some((t) => t.value === value) ? value! : DEFAULT_THEME;
}

export default function App() {
  return (
    <BrowserRouter>
      <NuqsAdapter>
        <ThemeProvider
          attribute='class'
          defaultTheme='system'
          enableSystem
          disableTransitionOnChange
          enableColorScheme
        >
          <ActiveThemeProvider initialTheme={readActiveTheme()}>
            <Providers>
              <Toaster />
              <AppRoutes />
            </Providers>
          </ActiveThemeProvider>
        </ThemeProvider>
      </NuqsAdapter>
    </BrowserRouter>
  );
}
