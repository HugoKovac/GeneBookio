// src/layouts/AuthLayout.tsx
import { Outlet } from 'react-router-dom';
import LanguageSwitcher from '../components/LanguageSwitcher';

export default function AuthLayout() {
  return (
    <div
      style={{
        minHeight: '100dvh',
        display: 'flex',
        flexDirection: 'column',
        paddingTop: 'env(safe-area-inset-top, 0px)',
        paddingBottom: 'env(safe-area-inset-bottom, 0px)',
        paddingLeft: 'env(safe-area-inset-left, 0px)',
        paddingRight: 'env(safe-area-inset-right, 0px)',
      }}
    >
      <LanguageSwitcher />
      <Outlet />
    </div>
  );
}
