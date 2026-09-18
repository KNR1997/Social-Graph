# React Shadcn Dashboard Starter

An admin dashboard starter built with **React 19 + Vite + React Router v7 + shadcn/ui + Tailwind v4 +
TypeScript**.

This is a port of [next-shadcn-dashboard-starter](../next-shadcn-dashboard-starter) to a pure React
SPA. No framework, no server: `vite build` emits static assets you can host anywhere.

## Quick start

```bash
npm install
cp .env.example .env      # defaults are fine for local development
npm run dev               # http://localhost:3000
```

The dashboard is gated behind authentication, served by the Go API in this
repository. Start it first (`make up && make migrate-up && make run` from the
repo root), then `npm run dev` and create an account at `/auth/sign-up`.

Vite proxies `/v1` to the API on :8080 so the browser sees a single origin.
That is required, not cosmetic: the session cookie is `SameSite=Lax`, so a
cross-origin setup would have the browser withhold it from every mutation.

| Script              | Does                                    |
| ------------------- | --------------------------------------- |
| `npm run dev`       | Vite dev server                         |
| `npm run build`     | typecheck, then production build        |
| `npm run preview`   | serve the built `dist/`                 |
| `npm run typecheck` | `tsc --noEmit`                          |
| `npm run lint`      | oxlint                                  |
| `npm run format`    | oxfmt                                   |

## Structure

```
src/
  main.tsx              entry; mounts App
  App.tsx               provider stack: Router > Nuqs > Theme > ActiveTheme > Query
  routes/               route tree + the auth guard
  layouts/              dashboard shell (sidebar, header, kbar, infobar)
  pages/                one component per route
  features/             feature modules: api/ + components/ + schemas/
  components/           ui/ (shadcn primitives), forms/, layout/, themes/, kbar/
  hooks/  lib/  config/  constants/  types/  styles/
```

Feature modules keep the layering of the original: `api/types.ts` → `api/service.ts` →
`api/queries.ts`. Components import from the service and query layers, never from the mocks directly,
so swapping in a real backend is a one-file change per feature (see the header comment in any
`service.ts`).

## Authentication

Session auth against the Go API. There is no auth SDK and no provider component:

- `POST /v1/auth/register` and `/login` set an **HttpOnly** session cookie. The token is never in
  the JSON body and no script can read it, which is the point of using a cookie rather than
  `localStorage`.
- `GET /v1/auth/me` is the whole client-side session state, held in one React Query entry
  (`sessionQueryOptions`). `useSession()` in `src/hooks/use-session.ts` is the `useUser` replacement.
- `routes/protected-route.tsx` guards the dashboard, `routes/guest-route.tsx` keeps signed-in users
  off the auth pages. Both wait out the pending state rather than treating "not known yet" as
  "signed out".
- `src/lib/api.ts` is the fetch wrapper. `credentials: 'include'` there is what sends the cookie;
  without it every request is anonymous.

The API destroys every other session on a password change and re-issues the caller's cookie.

## Data

Everything except auth still comes from in-memory [faker](https://fakerjs.dev) mocks in
`src/constants/mock-api.ts` and `mock-api-users.ts`. To point a feature at a real API, edit its
`api/service.ts` — `src/features/auth/api/service.ts` is a worked example of one that already is.

## What changed from the Next.js version

Most of the code is identical. The framework-specific pieces were replaced as follows.

| Next.js                                   | Here                                                       |
| ----------------------------------------- | ---------------------------------------------------------- |
| `src/app` file-based routing              | `src/routes/index.tsx` declarative route tree               |
| `layout.tsx`                              | `src/layouts/*` + layout routes                            |
| `loading.tsx`                             | `<Suspense fallback>`                                       |
| `error.tsx` / `not-found.tsx`             | `RouteErrorBoundary` (react-error-boundary)                 |
| parallel routes (`@area_stats`, …)        | suspending components in `features/overview/.../stat-slots` |
| `middleware` (`proxy.ts`)                 | `routes/protected-route.tsx`                                |
| `next/link` / `next/image`                | `components/link.tsx` / `components/image.tsx` shims        |
| `useRouter` / `usePathname`               | `hooks/use-router.ts` / `hooks/use-pathname.ts`             |
| `notFound()`                              | `lib/not-found.ts` (throws to the error boundary)           |
| `metadata` exports                        | `hooks/use-document-title.ts`; static tags in `index.html`  |
| `next/font/google`                        | `<link>` in `index.html` + `styles/fonts.css`               |
| `cookies()`                               | `lib/cookies.ts`, read on the client                        |
| server `prefetchQuery` + HydrationBoundary| `useSuspenseQuery` under a `<Suspense>`                     |
| `app/api` route handlers                  | dropped; services call the mocks directly                   |
| `NEXT_PUBLIC_*`                           | `VITE_*` (`import.meta.env`)                                |

The shims deliberately keep the Next-facing prop names (`Link href`, `Image fill`, `router.push`) so
components copied from the original compile unchanged.

`next-themes` was kept — it is framework-agnostic and drives light/dark mode here too.

### Things that genuinely differ in behavior

- **No SSR.** There is no server render, so no server prefetch and no streamed HTML. The overview
  page's staggered chart reveal is reproduced with per-card suspending queries.
- **No image optimization.** `components/image.tsx` renders a plain `<img>`.
- **The route guard is client-side.** It runs in the browser instead of in middleware. Treat it as
  UX, not a security boundary: the Go API enforces access on every request, which is the check that
  actually holds.
- **Lint.** The Next version's `'use client'` directives caused oxlint to skip its React Compiler
  rules on essentially the whole component tree. Without the directives those rules apply, so they
  are set to `warn` in `.oxlintrc.json`; see the comment there.

## Docs

Theme, form, and RBAC conventions carry over unchanged from the original project, whose `docs/`
folder still applies:

- `docs/forms.md` — TanStack Form + Zod, composable fields, multi-step, sheet/dialog forms
- `docs/themes.md` — OKLCH colors, adding a theme
- `docs/nav-rbac.md` — navigation access control

## Deployment

`npm run build` produces a static `dist/`. Deploy it to any static host — Vercel, Netlify, S3 +
CloudFront, GitHub Pages, nginx. Because this is a client-routed SPA, configure the host to rewrite
all unmatched paths to `/index.html`, or deep links like `/dashboard/product/12` will 404.

## Credit

Ported from [Kiranism/next-shadcn-dashboard-starter](https://github.com/Kiranism/next-shadcn-dashboard-starter).
