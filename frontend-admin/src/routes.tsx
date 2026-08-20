import { createBrowserRouter } from 'react-router-dom';
import MainLayout from './layouts/MainLayout';
import UploadPage from './pages/UploadPage';
import CataloguePage from './pages/CataloguePage';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <MainLayout />,
    children: [
      { index: true, element: <UploadPage /> },
      { path: 'catalogue', element: <CataloguePage /> },
    ],
  },
]);
