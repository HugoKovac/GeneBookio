import { Alert, Anchor, Button, PasswordInput, Stack, Text, TextInput } from '@mantine/core';
import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../features/auth/useAuth';
import { AuthCard } from './LoginPage';

export default function RegisterPage() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const password = String(form.get('password'));
    if (password !== String(form.get('confirmPassword'))) return setError('Passwords do not match.');
    setError(''); setLoading(true);
    try {
      await register({ firstname: String(form.get('firstname')), lastname: String(form.get('lastname')), email: String(form.get('email')), password });
      navigate('/dashboard', { replace: true });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to create your account.');
    } finally { setLoading(false); }
  }

  return <AuthCard title="Create your account" subtitle="Start building your personal library.">
    <form onSubmit={handleSubmit}><Stack>
      {error && <Alert color="red" radius="lg">{error}</Alert>}
      <TextInput name="firstname" label="First name" required autoComplete="given-name" size="md" radius="md" />
      <TextInput name="lastname" label="Last name" required autoComplete="family-name" size="md" radius="md" />
      <TextInput name="email" type="email" label="Email" required autoComplete="email" size="md" radius="md" />
      <PasswordInput name="password" label="Password" description="At least 12 characters" minLength={12} required autoComplete="new-password" size="md" radius="md" />
      <PasswordInput name="confirmPassword" label="Confirm password" minLength={12} required autoComplete="new-password" size="md" radius="md" />
      <Button type="submit" loading={loading} size="md" radius="xl" fullWidth mt="sm">Create account</Button>
      <Text size="sm" ta="center">Already have an account? <Anchor component={Link} to="/login">Sign in</Anchor></Text>
    </Stack></form>
  </AuthCard>;
}
