import { Button, Group, Paper, Stack, Text, Title } from '@mantine/core';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../features/auth/useAuth';

export default function DashboardPage() {
  const { logout } = useAuth();
  const navigate = useNavigate();
  return <Paper withBorder radius="md" p="xl" maw={680} mx="auto" mt={48}>
    <Stack><Title order={1}>Your library</Title><Text c="dimmed">You are signed in. This page is protected and is unavailable without an active session.</Text>
      <Group><Button onClick={() => navigate('/')}>Go home</Button><Button variant="subtle" color="red" onClick={() => { logout(); navigate('/login'); }}>Sign out</Button></Group>
    </Stack>
  </Paper>;
}
