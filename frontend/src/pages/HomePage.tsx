import { Alert, Anchor, Box, Button, Group, Loader, Stack, Text, ThemeIcon, Title } from '@mantine/core';
import { IconChevronRight, IconBook } from '@tabler/icons-react';
import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import BookCover from '../features/catalog/components/BookCover';
import { getBooks } from '../features/catalog/api';
import type { Book } from '../features/catalog/types';
import { getUser } from '../features/user/api';
import { useAuth } from '../features/auth/useAuth';

export default function HomePage() {
  const { isAuthenticated, userID, token } = useAuth();

  if (!isAuthenticated) {
    return <GuestHome />;
  }

  return <AuthenticatedHome userID={userID} token={token} />;
}

function GuestHome() {
  const { t } = useTranslation();

  return (
    <Stack justify="center" style={{ flex: 1, minHeight: 'calc(100dvh - 64px)' }} align="center" px="lg" gap="xl">
      <ThemeIcon size={72} radius="xl" variant="light" color="violet">
        <IconBook size={36} stroke={1.5} />
      </ThemeIcon>
      <Stack gap={6} align="center">
        <Title order={1} ta="center" style={{ fontSize: 32 }}>{t('app.name')}</Title>
        <Text c="dimmed" ta="center" maw={320}>{t('home.guest.tagline')}</Text>
      </Stack>
      <Stack w="100%" maw={340} gap="sm">
        <Button component={Link} to="/login" size="md" radius="xl" fullWidth>{t('home.guest.signIn')}</Button>
        <Button component={Link} to="/register" size="md" radius="xl" fullWidth variant="light">{t('home.guest.createAccount')}</Button>
      </Stack>
    </Stack>
  );
}

function AuthenticatedHome({ userID, token }: { userID: string | null; token: string | null }) {
  const { t } = useTranslation();
  const [firstname, setFirstname] = useState('');
  const [books, setBooks] = useState<Book[] | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!userID || !token) return;
    getUser(userID, token).then((user) => setFirstname(user.Firstname)).catch(() => {});
  }, [userID, token]);

  useEffect(() => {
    if (!token) return;
    getBooks(token).then(setBooks).catch((cause) => setError(cause instanceof Error ? cause.message : t('home.loadError')));
  }, [token, t]);

  return (
    <Stack px="lg" pt="lg" gap="xl">
      <Title order={2} style={{ fontSize: 28 }}>{firstname ? t('home.greeting', { name: firstname }) : t('home.welcomeBack')}</Title>

      <Stack gap="sm">
        <Group justify="space-between">
          <Text fw={600} size="lg">{t('home.continueLibrary')}</Text>
          <Anchor component={Link} to="/dashboard" size="sm" c="dimmed">
            <Group gap={2}>{t('home.seeAll')} <IconChevronRight size={14} /></Group>
          </Anchor>
        </Group>

        {error && <Alert color="red" radius="lg">{error}</Alert>}

        {!error && !books && <Group justify="center" py="xl"><Loader /></Group>}

        {books && books.length === 0 && (
          <Text c="dimmed">{t('home.noBooks')}</Text>
        )}

        {books && books.length > 0 && (
          <Box style={{ display: 'flex', gap: 12, overflowX: 'auto', paddingBottom: 8, scrollSnapType: 'x proximity' }}>
            {books.slice(0, 10).map((book) => (
              <Box key={book.ID} component={Link} to={`/books/${book.ID}`} style={{ flex: '0 0 120px', scrollSnapAlign: 'start', textDecoration: 'none', color: 'inherit' }}>
                <BookCover title={book.Title} coverURL={book.CoverURL} height={160} />
                <Text size="sm" fw={500} mt={6} lineClamp={2}>{book.Title}</Text>
              </Box>
            ))}
          </Box>
        )}
      </Stack>
    </Stack>
  );
}
