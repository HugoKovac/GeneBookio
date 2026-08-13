import { Alert, Anchor, Button, Paper, PasswordInput, Stack, Text, TextInput, Title } from '@mantine/core';
import { useState, type FormEvent } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../features/auth/useAuth';

export default function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setError('');
    setLoading(true);
    try {
      await login({ email: String(form.get('email')), password: String(form.get('password')) });
      navigate((location.state as { from?: { pathname?: string } } | null)?.from?.pathname ?? '/dashboard', { replace: true });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to sign in.');
    } finally {
      setLoading(false);
    }
  }

  return <AuthCard title="Welcome back" subtitle="Sign in to continue to your library.">
    <form onSubmit={handleSubmit}>
      <Stack>
        {error && <Alert color="red">{error}</Alert>}
        <TextInput name="email" type="email" label="Email" placeholder="you@example.com" required autoComplete="email" />
        <PasswordInput name="password" label="Password" minLength={12} required autoComplete="current-password" />
        <Button type="submit" loading={loading}>Sign in</Button>
        <Text size="sm" ta="center">New here? <Anchor component={Link} to="/register">Create an account</Anchor></Text>
      </Stack>
    </form>
  </AuthCard>;
}

export function AuthCard({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return <Paper withBorder shadow="md" radius="md" p="xl" maw={440} mx="auto" mt={48}>
    <Stack gap="lg">
      <div><Title order={2}>{title}</Title><Text c="dimmed" mt="xs">{subtitle}</Text></div>
      {children}
    </Stack>
  </Paper>;
}
