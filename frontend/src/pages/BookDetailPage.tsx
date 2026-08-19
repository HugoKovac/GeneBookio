import { Alert, Badge, Button, Group, Loader, Paper, Stack, Text, Title } from '@mantine/core';
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

  if (loadError) {
    return <Paper withBorder radius="md" p="xl" maw={680} mx="auto" mt={48}>
      <Stack>
        <Alert color="red">{loadError}</Alert>
        <Button variant="subtle" onClick={() => navigate('/dashboard')} w="fit-content">Back to library</Button>
      </Stack>
    </Paper>;
  }

  if (!book) {
    return <Group justify="center" mt={48}><Loader /></Group>;
  }

  return (
    <Paper withBorder radius="md" p="xl" maw={680} mx="auto" mt={48}>
      <Stack gap="lg">
        <Button variant="subtle" onClick={() => navigate('/dashboard')} w="fit-content" px={0}>&larr; Back to library</Button>

        <Group align="flex-start" wrap="nowrap">
          <div style={{ width: 140, flexShrink: 0 }}>
            <BookCover title={book.Title} coverURL={book.CoverURL} height={200} />
          </div>
          <Stack gap={4}>
            <Title order={2}>{book.Title}</Title>
            <Text c="dimmed">{(book.AuthorNames ?? []).join(', ') || 'Unknown author'}</Text>
            {book.ScriptGenerated && <Badge color="green" w="fit-content">Script ready</Badge>}
          </Stack>
        </Group>

        {book.Description && <Text>{book.Description}</Text>}

        <Stack gap="xs">
          <Title order={4}>Listen</Title>
          {audioLoading && <Loader size="sm" />}
          {audioError && <Alert color="yellow">{audioError}</Alert>}
          {audioURL && <audio controls src={audioURL} style={{ width: '100%' }} />}
        </Stack>
      </Stack>
    </Paper>
  );
}
