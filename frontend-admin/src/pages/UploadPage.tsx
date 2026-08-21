import { useEffect, useRef, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  FileInput,
  Group,
  Loader,
  Paper,
  ScrollArea,
  Select,
  Stack,
  Text,
  TextInput,
  Title,
} from '@mantine/core';
import { Link } from 'react-router-dom';
import BookCover from '../features/books/components/BookCover';
import { searchBooks, uploadBook } from '../features/books/api';
import type { Genre, Language, SearchResult } from '../features/books/types';

export default function UploadPage() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [selected, setSelected] = useState<SearchResult | null>(null);

  const [language, setLanguage] = useState<Language>('fr');
  const [genre, setGenre] = useState<Genre>('none-fiction');
  const [file, setFile] = useState<File | null>(null);

  const [status, setStatus] = useState<'idle' | 'uploading' | 'success' | 'error'>('idle');
  const [error, setError] = useState('');

  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    if (selected || query.trim().length === 0) {
      setResults([]);
      return;
    }

    const timer = setTimeout(() => {
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      setSearching(true);
      searchBooks(query.trim(), controller.signal)
        .then((books) => setResults(books ?? []))
        .catch((cause) => {
          if (cause instanceof DOMException && cause.name === 'AbortError') return;
          setResults([]);
        })
        .finally(() => setSearching(false));
    }, 250);

    return () => clearTimeout(timer);
  }, [query, selected]);

  function selectBook(book: SearchResult) {
    setSelected(book);
    setQuery(book.Title);
    setResults([]);
  }

  function clearSelection() {
    setSelected(null);
    setQuery('');
    setFile(null);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!selected || !file) return;

    setStatus('uploading');
    setError('');

    try {
      await uploadBook(selected.Key, language, genre, file);
      setStatus('success');
      clearSelection();
    } catch (cause) {
      setStatus('error');
      setError(cause instanceof Error ? cause.message : 'Upload failed.');
    }
  }

  return (
    <Stack maw={520} mx="auto" mt={24} gap="lg">
      <Title order={1}>Upload a book</Title>

      <div style={{ position: 'relative' }}>
        <TextInput
          label="Search Google Books"
          placeholder="Title, author..."
          value={query}
          onChange={(e) => {
            setQuery(e.currentTarget.value);
            if (selected) setSelected(null);
          }}
          rightSection={searching ? <Loader size="xs" /> : null}
          autoComplete="off"
        />

        {results.length > 0 && (
          <Paper withBorder shadow="md" pos="absolute" top="100%" left={0} right={0} mt={4} style={{ zIndex: 10 }}>
            <ScrollArea.Autosize mah={300}>
              <Stack gap={0}>
                {results.map((book) => (
                  <Group
                    key={book.Key}
                    wrap="nowrap"
                    gap="sm"
                    p="xs"
                    style={{ cursor: 'pointer' }}
                    onClick={() => selectBook(book)}
                  >
                    <div style={{ width: 32, flexShrink: 0 }}>
                      <BookCover title={book.Title} coverURL={book.CoverURL} height={48} />
                    </div>
                    <Stack gap={0}>
                      <Text size="sm" fw={600} lineClamp={1}>{book.Title}</Text>
                      <Text size="xs" c="dimmed" lineClamp={1}>{(book.AuthorNames ?? []).join(', ')}</Text>
                    </Stack>
                  </Group>
                ))}
              </Stack>
            </ScrollArea.Autosize>
          </Paper>
        )}
      </div>

      {selected && (
        <Card withBorder padding="sm">
          <Group wrap="nowrap">
            <div style={{ width: 48, flexShrink: 0 }}>
              <BookCover title={selected.Title} coverURL={selected.CoverURL} height={72} />
            </div>
            <Stack gap={0} style={{ flex: 1 }}>
              <Text fw={600} lineClamp={2}>{selected.Title}</Text>
              <Text size="sm" c="dimmed" lineClamp={1}>{(selected.AuthorNames ?? []).join(', ')}</Text>
            </Stack>
            <Button variant="subtle" color="red" onClick={clearSelection}>Remove</Button>
          </Group>
        </Card>
      )}

      <form onSubmit={handleSubmit}>
        <Stack gap="md">
          <Select
            label="Language"
            data={[
              { value: 'fr', label: 'French' },
              { value: 'en', label: 'English' },
            ]}
            value={language}
            onChange={(value) => setLanguage((value as Language) ?? 'fr')}
            allowDeselect={false}
          />

          <Select
            label="Genre"
            data={[
              { value: 'none-fiction', label: 'Non-fiction' },
              { value: 'fiction', label: 'Fiction' },
            ]}
            value={genre}
            onChange={(value) => setGenre((value as Genre) ?? 'none-fiction')}
            allowDeselect={false}
          />

          <FileInput
            label="EPUB file"
            placeholder="Select a .epub file"
            accept="application/epub+zip,.epub"
            value={file}
            onChange={setFile}
            required
          />

          {status === 'success' && (
            <Alert color="green">
              Book uploaded and queued for processing. Track its progress on the{' '}
              <Text component={Link} to="/catalogue" fw={600} span>catalogue</Text> page.
            </Alert>
          )}
          {status === 'error' && <Alert color="red">{error}</Alert>}

          <Button type="submit" disabled={!selected || !file} loading={status === 'uploading'}>
            Upload
          </Button>
        </Stack>
      </form>
    </Stack>
  );
}
