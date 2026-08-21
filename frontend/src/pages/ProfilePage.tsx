import { Alert, Avatar, Button, Group, Loader, Modal, Paper, Stack, Text, TextInput, Title, UnstyledButton } from '@mantine/core';
import { IconChevronRight, IconCreditCard, IconLogout, IconTrash } from '@tabler/icons-react';
import { Capacitor } from '@capacitor/core';
import { useDisclosure } from '@mantine/hooks';
import { useEffect, useState, type FormEvent, type ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../features/auth/useAuth';
import { createPortalSession } from '../features/subscription/api';
import { useSubscription } from '../features/subscription/useSubscription';
import { deleteUser, getUser, updateUser } from '../features/user/api';
import type { User } from '../features/user/types';

// A native purchaser has no Stripe customer to manage — send them to the
// OS's own subscription-management surface instead (Apple/Google expect
// in-app access to manage or cancel, not just a hidden row).
const nativeSubscriptionManagementURL = Capacitor.getPlatform() === 'ios'
  ? 'itms-apps://apps.apple.com/account/subscriptions'
  : 'https://play.google.com/store/account/subscriptions';

export default function ProfilePage() {
  const { userID, token, logout } = useAuth();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { isActive: hasActiveSubscription } = useSubscription();

  const [user, setUser] = useState<User | null>(null);
  const [loadError, setLoadError] = useState('');

  const [portalError, setPortalError] = useState('');
  const [portalLoading, setPortalLoading] = useState(false);

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
      .catch((cause) => setLoadError(cause instanceof Error ? cause.message : t('profile.loadError')));
  }, [userID, token, t]);

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
      setSaveError(cause instanceof Error ? cause.message : t('profile.saveError'));
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
      setDeleteError(cause instanceof Error ? cause.message : t('profile.deleteError'));
      setDeleting(false);
    }
  }

  function handleLogout() {
    logout();
    navigate('/login', { replace: true });
  }

  async function handleManageSubscription() {
    if (Capacitor.isNativePlatform()) {
      window.open(nativeSubscriptionManagementURL, '_system');
      return;
    }
    if (!token) return;
    setPortalError('');
    setPortalLoading(true);
    try {
      const { portalUrl } = await createPortalSession(token);
      window.location.href = portalUrl;
    } catch (cause) {
      setPortalError(cause instanceof Error ? cause.message : t('profile.portalError'));
      setPortalLoading(false);
    }
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
      <Title order={2} style={{ fontSize: 28 }}>{t('profile.title')}</Title>

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
            {saveSuccess && <Alert color="green" radius="lg">{t('profile.updated')}</Alert>}
            <TextInput name="firstname" label={t('profile.firstname')} defaultValue={user.Firstname} required maxLength={100} autoComplete="given-name" radius="md" />
            <TextInput name="lastname" label={t('profile.lastname')} defaultValue={user.Lastname} required maxLength={100} autoComplete="family-name" radius="md" />
            <Button type="submit" loading={saving} radius="xl">{t('profile.save')}</Button>
          </Stack>
        </form>
      </Paper>

      {portalError && <Alert color="red" radius="lg">{portalError}</Alert>}

      <Paper withBorder radius="lg" style={{ overflow: 'hidden' }}>
        {hasActiveSubscription && (
          <SettingsRow icon={<IconCreditCard size={20} />} label={t('profile.manageSubscription')} onClick={handleManageSubscription} disabled={portalLoading} />
        )}
        <SettingsRow icon={<IconLogout size={20} />} label={t('profile.signOut')} onClick={handleLogout} />
        <SettingsRow icon={<IconTrash size={20} />} label={t('profile.deleteAccount')} onClick={openConfirm} color="red" last />
      </Paper>

      <Modal opened={confirmOpened} onClose={closeConfirm} title={t('profile.deleteAccount')} centered radius="lg">
        <Stack>
          {deleteError && <Alert color="red" radius="lg">{deleteError}</Alert>}
          <Text>{t('profile.deleteAccountBody')}</Text>
          <Group justify="flex-end">
            <Button variant="default" radius="xl" onClick={closeConfirm}>{t('profile.cancel')}</Button>
            <Button color="red" radius="xl" loading={deleting} onClick={handleDelete}>{t('profile.deleteConfirm')}</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}

function SettingsRow({ icon, label, onClick, color, last, disabled }: { icon: ReactNode; label: string; onClick: () => void; color?: string; last?: boolean; disabled?: boolean }) {
  return (
    <UnstyledButton
      onClick={onClick}
      disabled={disabled}
      style={{ display: 'block', width: '100%', borderBottom: last ? undefined : '1px solid var(--mantine-color-default-border)', opacity: disabled ? 0.6 : 1 }}
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
