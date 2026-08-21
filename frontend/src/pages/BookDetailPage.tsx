import { ActionIcon, Alert, Badge, Box, Group, Loader, Stack, Text, Title, UnstyledButton } from '@mantine/core';
import { IconChevronLeft, IconPlayerPauseFilled, IconPlayerPlayFilled } from '@tabler/icons-react';
import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import BookCover from '../features/catalog/components/BookCover';
import { getBookAudioStreamURL, getBooks } from '../features/catalog/api';
import type { Book } from '../features/catalog/types';
import { useAuth } from '../features/auth/useAuth';
import { usePlayer } from '../features/player/usePlayer';

export default function BookDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { token } = useAuth();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { track, isPlaying, playTrack, togglePlay, error: playerError } = usePlayer();

  const [book, setBook] = useState<Book | null>(null);
  const [loadError, setLoadError] = useState('');

  useEffect(() => {
    if (!token || !id) return;
    getBooks(token)
      .then((allBooks) => {
        const found = allBooks.find((b) => b.ID === id);
        setBook(found ?? null);
        if (!found) setLoadError(t('bookDetail.notFound'));
      })
      .catch((cause) => setLoadError(cause instanceof Error ? cause.message : t('bookDetail.loadError')));
  }, [id, token, t]);

  const isCurrentTrack = track?.bookID === id;
  const audioError = isCurrentTrack && playerError;

  const handlePlay = () => {
    if (!book || !id || !token) return;
    if (isCurrentTrack) {
      togglePlay();
      return;
    }
    playTrack({
      bookID: id,
      title: book.Title,
      author: (book.AuthorNames ?? []).join(', ') || t('bookDetail.unknownAuthor'),
      coverURL: book.CoverURL,
      src: getBookAudioStreamURL(id, token),
    });
  };

  const backButton = (
    <ActionIcon variant="subtle" color="gray" size="lg" radius="xl" onClick={() => navigate('/dashboard')} aria-label={t('bookDetail.back')}>
      <IconChevronLeft size={22} />
    </ActionIcon>
  );

  if (loadError) {
    return <Stack px="lg" pt="lg" gap="lg">
      {backButton}
      <Alert color="red" radius="lg">{loadError}</Alert>
    </Stack>;
  }

  if (!book) {
    return <Group justify="center" mt={48}><Loader /></Group>;
  }

  return (
    <Stack px="lg" pt="lg" pb="xl" gap="xl">
      {backButton}

      <Stack align="center" gap="md">
        <Box w={200}>
          <BookCover title={book.Title} coverURL={book.CoverURL} height={260} />
        </Box>
        <Stack gap={4} align="center">
          <Title order={2} ta="center">{book.Title}</Title>
          <Text c="dimmed" ta="center">{(book.AuthorNames ?? []).join(', ') || t('bookDetail.unknownAuthor')}</Text>
          {book.ScriptGenerated && <Badge color="green" variant="light" mt={4}>{t('bookDetail.scriptReady')}</Badge>}
        </Stack>
      </Stack>

      <Stack gap="xs" align="center">
        {audioError ? (
          <Alert color="yellow" radius="lg" w="100%">{t('bookDetail.summaryUnavailable')}</Alert>
        ) : (
          <UnstyledButton
            onClick={handlePlay}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: 8,
              padding: '10px 28px',
              borderRadius: 999,
              background: 'var(--mantine-primary-color-filled)',
              color: 'white',
            }}
          >
            {isCurrentTrack && isPlaying ? <IconPlayerPauseFilled size={18} /> : <IconPlayerPlayFilled size={18} />}
            <Text fw={600} c="white">
              {isCurrentTrack && isPlaying ? t('bookDetail.pause') : t('bookDetail.play')}
            </Text>
          </UnstyledButton>
        )}
      </Stack>

      {book.Description && (
        <Stack gap={4}>
          <Text fw={600}>{t('bookDetail.description')}</Text>
          <Text c="dimmed">{book.Description}</Text>
        </Stack>
      )}
    </Stack>
  );
}
