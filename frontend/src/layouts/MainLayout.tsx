// src/layouts/MainLayout.tsx
import { Outlet, Link, useLocation } from 'react-router-dom';
import { AppShell, Burger, Button, Group, NavLink, Title } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { useAuth } from '../features/auth/useAuth';

export default function MainLayout() {
  const [opened, { toggle }] = useDisclosure();
  const location = useLocation();
  const { isAuthenticated, logout } = useAuth();

  return (
    <AppShell
      header={{ height: 60 }}
      navbar={{ width: 200, breakpoint: 'sm', collapsed: { mobile: !opened } }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md">
          <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
          <Title order={3}>GeneBookio</Title>
          <Group ml="auto">
            {isAuthenticated ? <Button variant="subtle" onClick={logout}>Sign out</Button> : <Button component={Link} to="/login">Sign in</Button>}
          </Group>
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="md">
        <NavLink
          component={Link}
          to="/"
          label="Home"
          active={location.pathname === '/'}
        />
        {isAuthenticated && <NavLink component={Link} to="/dashboard" label="My library" active={location.pathname === '/dashboard'} />}
        <NavLink
          component={Link}
          to="/about"
          label="About"
          active={location.pathname === '/about'}
        />
      </AppShell.Navbar>

      <AppShell.Main>
        <Outlet />
      </AppShell.Main>
    </AppShell>
  );
}
