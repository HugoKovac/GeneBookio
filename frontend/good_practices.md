# Project Architecture Guide

A common, battle-tested way to organize a React + TypeScript project (e.g. Vite + Mantine + React Router).

## Folder Structure

```
src/
├── pages/              # One page = one route
├── components/         # Reusable UI components (pure UI)
├── features/           # Business modules (optional, for larger projects)
├── layouts/             # Layout wrappers (AppShell, etc.)
├── hooks/              # Reusable custom hooks
├── lib/ or utils/      # Utility functions, helpers
├── api/ or services/   # Network calls, fetch, axios config...
├── types/               # Shared types/interfaces
├── routes.tsx           # Centralized route configuration
└── App.tsx
```

## Units and Their Roles

### `pages/`
A page = an entry point for a route (`HomePage.tsx`, `AboutPage.tsx`, `UserProfilePage.tsx`). A page orchestrates: it assembles components, handles high-level data fetching, and connects query params. It shouldn't contain complex UI logic itself.

### `components/`
Reusable, "dumb" building blocks as much as possible: `Button`, `UserCard`, `ProductList`. Ideally stateless or with very local state (e.g. a dropdown open/closed). They shouldn't know about routing or direct API calls.

### `layouts/`
Structural wrappers (header, sidebar, footer) like an `AppShell`. A layout wraps pages via `<Outlet />` when using React Router's nested routes.

### `features/` (optional, but recommended once the project grows)
Groups code by business domain rather than by technical type: `features/auth/`, `features/dashboard/`, each with its own components, hooks, types, and local api calls. Avoids ending up with a catch-all `components/` folder with 50 files.

## Concrete Example

```
src/
├── layouts/
│   └── MainLayout.tsx        # AppShell + Outlet
├── pages/
│   ├── HomePage.tsx
│   └── AboutPage.tsx
├── features/
│   └── users/
│       ├── components/
│       │   └── UserCard.tsx
│       ├── hooks/
│       │   └── useUsers.ts
│       ├── api.ts
│       └── types.ts
├── components/                # generic shared components (custom buttons, etc.)
├── routes.tsx
└── App.tsx
```

## Simple Decision Rule

| Criteria | Location |
|---|---|
| Used by a single route, orchestrates the rest | `pages/` |
| Reused in multiple places, no business logic | `components/` |
| Tied to a specific business domain (auth, cart, users...) | `features/<domain>/` |
| Repeated wrapping structure across multiple pages | `layouts/` |

## Note on Project Size

For a small project (like a starter template), you can stick to `pages/` + `components/` + `layouts/` without `features/` — add it only once the project grows and `components/` becomes unmanageable.
