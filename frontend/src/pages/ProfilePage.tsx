import { Alert, Avatar, Button, Group, Loader, Modal, Paper, Stack, Text, TextInput, Title, UnstyledButton } from '@mantine/core';
import { IconChevronRight, IconLogout, IconTrash } from '@tabler/icons-react';
import { useDisclosure } from '@mantine/hooks';
import { useEffect, useState, type FormEvent, type ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../features/auth/useAuth';
import { deleteUser, getUser, updateUser } from '../features/user/api';
import type { User } from '../features/user/types';

export default function ProfilePage() {
  const { userID, token, logout } = useAuth();
  const navigate = useNavigate();

  const [user, setUser] = useState<User | null>(null);
  const [loadError, setLoadError] = useState('');

  const [saveError, setSaveError] = useState('');
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [saving, setSaving] = useState(false);

  const [deleteError, setDeleteError] = useState('');
  const [deleting, setDeleting] = useState(false);
  const [confirmOpened, { open: openConfirm, close: closeConfirm }] = useDisclosure(false);

  useEffect(() => {
    if (!userID || !token) return;
    getUser(userID, token)
      .then(setUser)
      .catch((cause) => setLoadError(cause instanceof Error ? cause.message : 'Unable to load your profile.'));
  }, [userID, token]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!userID || !token) return;

    const form = new FormData(event.currentTarget);
    setSaveError('');
    setSaveSuccess(false);
    setSaving(true);
    try {
      const updated = await updateUser(userID, token, {
        firstname: String(form.get('firstname')),
        lastname: String(form.get('lastname')),
      });
      setUser(updated);
      setSaveSuccess(true);
    } catch (cause) {
      setSaveError(cause instanceof Error ? cause.message : 'Unable to update your profile.');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!userID || !token) return;
    setDeleteError('');
    setDeleting(true);
    try {
      await deleteUser(userID, token);
      closeConfirm();
      logout();
      navigate('/', { replace: true });
    } catch (cause) {
      setDeleteError(cause instanceof Error ? cause.message : 'Unable to delete your account.');
      setDeleting(false);
    }
  }

  function handleLogout() {
    logout();
    navigate('/login', { replace: true });
  }

  if (loadError) {
    return <Stack px="lg" pt="lg"><Alert color="red" radius="lg">{loadError}</Alert></Stack>;
  }

  if (!user) {
    return <Group justify="center" mt={48}><Loader /></Group>;
  }

  const initials = `${user.Firstname[0] ?? ''}${user.Lastname[0] ?? ''}`.toUpperCase();

  return (
    <Stack px="lg" pt="lg" pb="xl" gap="xl">
      <Title order={2} style={{ fontSize: 28 }}>Profile</Title>

      <Group>
        <Avatar size={64} radius="xl" color="violet">{initials}</Avatar>
        <Stack gap={0}>
          <Text fw={600} size="lg">{user.Firstname} {user.Lastname}</Text>
          <Text c="dimmed" size="sm">{user.Email}</Text>
        </Stack>
      </Group>

      <Paper withBorder radius="lg" p="md">
        <form onSubmit={handleSubmit}>
          <Stack>
            {saveError && <Alert color="red" radius="lg">{saveError}</Alert>}
            {saveSuccess && <Alert color="green" radius="lg">Your profile has been updated.</Alert>}
            <TextInput name="firstname" label="First name" defaultValue={user.Firstname} required maxLength={100} autoComplete="given-name" radius="md" />
            <TextInput name="lastname" label="Last name" defaultValue={user.Lastname} required maxLength={100} autoComplete="family-name" radius="md" />
            <Button type="submit" loading={saving} radius="xl">Save changes</Button>
          </Stack>
        </form>
      </Paper>

      <Paper withBorder radius="lg" style={{ overflow: 'hidden' }}>
        <SettingsRow icon={<IconLogout size={20} />} label="Sign out" onClick={handleLogout} />
        <SettingsRow icon={<IconTrash size={20} />} label="Delete account" onClick={openConfirm} color="red" last />
      </Paper>

      <Modal opened={confirmOpened} onClose={closeConfirm} title="Delete account" centered radius="lg">
        <Stack>
          {deleteError && <Alert color="red" radius="lg">{deleteError}</Alert>}
          <Text>This will permanently delete your account. You will no longer be able to sign in. This action cannot be undone.</Text>
          <Group justify="flex-end">
            <Button variant="default" radius="xl" onClick={closeConfirm}>Cancel</Button>
            <Button color="red" radius="xl" loading={deleting} onClick={handleDelete}>Delete my account</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}

function SettingsRow({ icon, label, onClick, color, last }: { icon: ReactNode; label: string; onClick: () => void; color?: string; last?: boolean }) {
  return (
    <UnstyledButton
      onClick={onClick}
      style={{ display: 'block', width: '100%', borderBottom: last ? undefined : '1px solid var(--mantine-color-default-border)' }}
    >
      <Group justify="space-between" px="md" py="sm">
        <Group gap="sm" c={color}>
          {icon}
          <Text c={color}>{label}</Text>
        </Group>
        <IconChevronRight size={18} color="var(--mantine-color-dimmed)" />
      </Group>
    </UnstyledButton>
  );
}
