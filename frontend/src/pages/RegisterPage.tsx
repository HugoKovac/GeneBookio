import { Alert, Anchor, Button, PasswordInput, Stack, Text, TextInput } from '@mantine/core';
import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../features/auth/useAuth';
import { AuthCard } from './LoginPage';

export default function RegisterPage() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const password = String(form.get('password'));
    if (password !== String(form.get('confirmPassword'))) return setError(t('auth.register.passwordMismatch'));
    setError(''); setLoading(true);
    try {
      const language = navigator.language.toLowerCase().startsWith('fr') ? 'fr' : 'en';
      await register({ firstname: String(form.get('firstname')), lastname: String(form.get('lastname')), email: String(form.get('email')), password, language });
      navigate('/dashboard', { replace: true });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t('auth.register.error'));
    } finally { setLoading(false); }
  }

  return <AuthCard title={t('auth.register.title')} subtitle={t('auth.register.subtitle')}>
    <form onSubmit={handleSubmit}><Stack>
      {error && <Alert color="red" radius="lg">{error}</Alert>}
      <TextInput name="firstname" label={t('auth.register.firstname')} required autoComplete="given-name" size="md" radius="md" />
      <TextInput name="lastname" label={t('auth.register.lastname')} required autoComplete="family-name" size="md" radius="md" />
      <TextInput name="email" type="email" label={t('auth.register.email')} required autoComplete="email" size="md" radius="md" />
      <PasswordInput name="password" label={t('auth.register.password')} description={t('auth.register.passwordHint')} minLength={12} required autoComplete="new-password" size="md" radius="md" />
      <PasswordInput name="confirmPassword" label={t('auth.register.confirmPassword')} minLength={12} required autoComplete="new-password" size="md" radius="md" />
      <Button type="submit" loading={loading} size="md" radius="xl" fullWidth mt="sm">{t('auth.register.submit')}</Button>
      <Text size="sm" ta="center">{t('auth.register.haveAccount')} <Anchor component={Link} to="/login">{t('auth.register.signIn')}</Anchor></Text>
    </Stack></form>
  </AuthCard>;
}
