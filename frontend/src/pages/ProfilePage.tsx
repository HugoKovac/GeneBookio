import { Alert, Button, Group, Loader, Modal, Paper, Stack, Text, TextInput, Title } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { useEffect, useState, type FormEvent } from 'react';
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
    return <Paper withBorder radius="md" p="xl" maw={480} mx="auto" mt={48}>
      <Alert color="red">{loadError}</Alert>
    </Paper>;
  }

  if (!user) {
    return <Group justify="center" mt={48}><Loader /></Group>;
  }

  return <Paper withBorder radius="md" p="xl" maw={480} mx="auto" mt={48}>
    <Stack gap="lg">
      <div>
        <Title order={2}>Your profile</Title>
        <Text c="dimmed" mt="xs">Update your account details.</Text>
      </div>

      <form onSubmit={handleSubmit}>
        <Stack>
          {saveError && <Alert color="red">{saveError}</Alert>}
          {saveSuccess && <Alert color="green">Your profile has been updated.</Alert>}
          <TextInput label="Email" value={user.Email} disabled />
          <TextInput name="firstname" label="First name" defaultValue={user.Firstname} required maxLength={100} autoComplete="given-name" />
          <TextInput name="lastname" label="Last name" defaultValue={user.Lastname} required maxLength={100} autoComplete="family-name" />
          <Button type="submit" loading={saving}>Save changes</Button>
        </Stack>
      </form>

      <Stack gap="xs">
        <Button variant="subtle" onClick={handleLogout}>Sign out</Button>
        <Button variant="subtle" color="red" onClick={openConfirm}>Delete account</Button>
      </Stack>
    </Stack>

    <Modal opened={confirmOpened} onClose={closeConfirm} title="Delete account" centered>
      <Stack>
        {deleteError && <Alert color="red">{deleteError}</Alert>}
        <Text>This will permanently delete your account. You will no longer be able to sign in. This action cannot be undone.</Text>
        <Group justify="flex-end">
          <Button variant="default" onClick={closeConfirm}>Cancel</Button>
          <Button color="red" loading={deleting} onClick={handleDelete}>Delete my account</Button>
        </Group>
      </Stack>
    </Modal>
  </Paper>;
}
