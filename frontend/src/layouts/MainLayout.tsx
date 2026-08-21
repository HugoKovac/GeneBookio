// src/layouts/MainLayout.tsx
import { Outlet, Link, useLocation } from 'react-router-dom';
import { AppShell, Text, UnstyledButton } from '@mantine/core';
import { IconHome, IconHomeFilled, IconHeadphones, IconHeadphonesFilled, IconUser, IconUserFilled } from '@tabler/icons-react';
import type { TablerIcon } from '@tabler/icons-react';
import { useAuth } from '../features/auth/useAuth';

type Tab = {
  to: string;
  label: string;
  icon: TablerIcon;
  activeIcon: TablerIcon;
  isActive: (pathname: string) => boolean;
};

export default function MainLayout() {
  const location = useLocation();
  const { isAuthenticated } = useAuth();

  const tabs: Tab[] = [
    { to: '/', label: 'Home', icon: IconHome, activeIcon: IconHomeFilled, isActive: (p) => p === '/' },
    { to: '/dashboard', label: 'Library', icon: IconHeadphones, activeIcon: IconHeadphonesFilled, isActive: (p) => p.startsWith('/dashboard') || p.startsWith('/books') },
    {
      to: isAuthenticated ? '/profile' : '/login',
      label: isAuthenticated ? 'Profile' : 'Sign in',
      icon: IconUser,
      activeIcon: IconUserFilled,
      isActive: (p) => p === '/profile' || p === '/login' || p === '/register',
    },
  ];

  return (
    <AppShell footer={{ height: 64 }}>
      <AppShell.Main pt="env(safe-area-inset-top, 0px)" pl="env(safe-area-inset-left, 0px)" pr="env(safe-area-inset-right, 0px)">
        <Outlet />
      </AppShell.Main>

      <AppShell.Footer>
        <nav style={{ display: 'flex', height: 64 }}>
          {tabs.map((tab) => {
            const active = tab.isActive(location.pathname);
            const Icon = active ? tab.activeIcon : tab.icon;
            return (
              <UnstyledButton
                key={tab.label}
                component={Link}
                to={tab.to}
                style={{
                  flex: 1,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: 2,
                  color: active ? 'var(--accent)' : 'var(--text)',
                }}
              >
                <Icon size={24} stroke={1.75} />
                <Text size="xs" fw={active ? 600 : 400}>{tab.label}</Text>
              </UnstyledButton>
            );
          })}
        </nav>
      </AppShell.Footer>
    </AppShell>
  );
}
