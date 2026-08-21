import { Alert, Anchor, Button, PasswordInput, Stack, Text, TextInput, ThemeIcon, Title } from '@mantine/core';
import { IconBook } from '@tabler/icons-react';
import { useState, type FormEvent, type ReactNode } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../features/auth/useAuth';

export default function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const { t } = useTranslation();
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
      setError(cause instanceof Error ? cause.message : t('auth.login.error'));
    } finally {
      setLoading(false);
    }
  }

  return <AuthCard title={t('auth.login.title')} subtitle={t('auth.login.subtitle')}>
    <form onSubmit={handleSubmit}>
      <Stack>
        {error && <Alert color="red" radius="lg">{error}</Alert>}
        <TextInput name="email" type="email" label={t('auth.login.email')} placeholder="you@example.com" required autoComplete="email" size="md" radius="md" />
        <PasswordInput name="password" label={t('auth.login.password')} minLength={12} required autoComplete="current-password" size="md" radius="md" />
        <Button type="submit" loading={loading} size="md" radius="xl" fullWidth mt="sm">{t('auth.login.submit')}</Button>
        <Text size="sm" ta="center">{t('auth.login.noAccount')} <Anchor component={Link} to="/register">{t('auth.login.createAccount')}</Anchor></Text>
      </Stack>
    </form>
  </AuthCard>;
}

export function AuthCard({ title, subtitle, children }: { title: string; subtitle: string; children: ReactNode }) {
  return (
    <Stack justify="center" style={{ flex: 1 }} w="100%" maw={420} mx="auto" px="lg" py="xl" gap="xl">
      <Stack gap={6} align="center">
        <ThemeIcon size={64} radius="xl" variant="light" color="violet">
          <IconBook size={32} stroke={1.5} />
        </ThemeIcon>
        <Title order={2} mt="sm" ta="center">{title}</Title>
        <Text c="dimmed" ta="center">{subtitle}</Text>
      </Stack>
      {children}
    </Stack>
  );
}
