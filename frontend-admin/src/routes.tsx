import { createBrowserRouter } from 'react-router-dom';
import MainLayout from './layouts/MainLayout';
import UploadPage from './pages/UploadPage';
import CataloguePage from './pages/CataloguePage';

export const router = createBrowserRouter(
  [
    {
      path: '/',
      element: <MainLayout />,
      children: [
        { index: true, element: <UploadPage /> },
        { path: 'catalogue', element: <CataloguePage /> },
      ],
    },
  ],
  // Matches Vite's `base` (see vite.config.ts) so links/navigation stay correct
  // when this app is served under a path prefix (e.g. /admin/).
  { basename: import.meta.env.BASE_URL },
);
