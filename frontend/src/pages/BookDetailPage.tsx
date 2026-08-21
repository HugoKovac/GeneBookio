import { ActionIcon, Alert, Badge, Box, Group, Loader, Stack, Text, Title } from '@mantine/core';
import { IconChevronLeft } from '@tabler/icons-react';
import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import BookCover from '../features/catalog/components/BookCover';
import { getBookAudioURL, getBooks } from '../features/catalog/api';
import type { Book } from '../features/catalog/types';
import { useAuth } from '../features/auth/useAuth';

export default function BookDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { token } = useAuth();
  const navigate = useNavigate();

  const [book, setBook] = useState<Book | null>(null);
  const [loadError, setLoadError] = useState('');

  const [audioURL, setAudioURL] = useState('');
  const [audioError, setAudioError] = useState('');
  const [audioLoading, setAudioLoading] = useState(true);

  useEffect(() => {
    if (!token || !id) return;
    getBooks(token)
      .then((allBooks) => {
        const found = allBooks.find((b) => b.ID === id);
        setBook(found ?? null);
        if (!found) setLoadError('This book could not be found in your library.');
      })
      .catch((cause) => setLoadError(cause instanceof Error ? cause.message : 'Unable to load this book.'));
  }, [id, token]);

  useEffect(() => {
    if (!token || !id) return;

    let objectURL = '';
    setAudioLoading(true);
    setAudioError('');

    getBookAudioURL(id, token)
      .then((url) => {
        objectURL = url;
        setAudioURL(url);
      })
      .catch(() => setAudioError('Audio is not available for this book yet.'))
      .finally(() => setAudioLoading(false));

    return () => {
      if (objectURL) URL.revokeObjectURL(objectURL);
    };
  }, [id, token]);

  const backButton = (
    <ActionIcon variant="subtle" color="gray" size="lg" radius="xl" onClick={() => navigate('/dashboard')} aria-label="Back to library">
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
          <Text c="dimmed" ta="center">{(book.AuthorNames ?? []).join(', ') || 'Unknown author'}</Text>
          {book.ScriptGenerated && <Badge color="green" variant="light" mt={4}>Script ready</Badge>}
        </Stack>
      </Stack>

      <Stack gap="xs">
        {audioLoading && <Group justify="center"><Loader size="sm" /></Group>}
        {audioError && <Alert color="yellow" radius="lg">{audioError}</Alert>}
        {audioURL && <audio controls src={audioURL} style={{ width: '100%' }} />}
      </Stack>

      {book.Description && (
        <Stack gap={4}>
          <Text fw={600}>Description</Text>
          <Text c="dimmed">{book.Description}</Text>
        </Stack>
      )}
    </Stack>
  );
}
