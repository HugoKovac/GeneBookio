import { Alert, Card, Group, Loader, SimpleGrid, Stack, Text, Title } from '@mantine/core';
import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import BookCover from '../features/catalog/components/BookCover';
import { getBooks } from '../features/catalog/api';
import type { Book } from '../features/catalog/types';
import { useAuth } from '../features/auth/useAuth';

export default function DashboardPage() {
  const { token } = useAuth();

  const [books, setBooks] = useState<Book[] | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!token) return;
    getBooks(token)
      .then(setBooks)
      .catch((cause) => setError(cause instanceof Error ? cause.message : 'Unable to load your library.'));
  }, [token]);

  if (error) {
    return <Alert color="red" radius="lg" m="lg">{error}</Alert>;
  }

  if (!books) {
    return <Group justify="center" mt={48}><Loader /></Group>;
  }

  return (
    <Stack px="lg" pt="lg" pb="lg" gap="lg">
      <Title order={2} style={{ fontSize: 28 }}>Your Library</Title>

      {books.length === 0
        ? <Text c="dimmed">No books in your catalog yet.</Text>
        : <SimpleGrid cols={{ base: 2, sm: 3, md: 4 }} spacing="md">
            {books.map((book) => (
              <Card key={book.ID} component={Link} to={`/books/${book.ID}`} radius="lg" padding="xs" shadow="sm">
                <Card.Section>
                  <BookCover title={book.Title} coverURL={book.CoverURL} />
                </Card.Section>
                <Stack gap={2} mt="sm" px={2} pb={2}>
                  <Text fw={600} size="sm" lineClamp={2}>{book.Title}</Text>
                  <Text size="xs" c="dimmed" lineClamp={1}>{(book.AuthorNames ?? []).join(', ') || 'Unknown author'}</Text>
                </Stack>
              </Card>
            ))}
          </SimpleGrid>}
    </Stack>
  );
}
