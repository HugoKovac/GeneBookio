import { useEffect, useState } from 'react';
import { ActionIcon, Alert, Badge, Box, Button, Card, Group, Loader, Stack, Table, Text, Title, Tooltip } from '@mantine/core';
import BookCover from '../features/books/components/BookCover';
import { getCatalog, retryBook } from '../features/books/api';
import type { CatalogBook } from '../features/books/types';

const STAGES: { key: keyof CatalogBook; label: string }[] = [
  { key: 'Uploaded', label: 'Uploaded' },
  { key: 'Parsed', label: 'Parsed' },
  { key: 'Prepared', label: 'Prepared' },
  { key: 'ScriptGenerated', label: 'Script' },
  { key: 'TTSGenerated', label: 'Audio' },
];

// Maps a failed book's FailedStage (the backend queue channel that failed,
// e.g. "prepare") to the progress dot it was working towards when it failed.
const FAILED_STAGE_TO_PROGRESS_KEY: Record<string, keyof CatalogBook> = {
  split: 'Parsed',
  prepare: 'Prepared',
  generate_script: 'ScriptGenerated',
  generate_tts: 'TTSGenerated',
};

const POLL_INTERVAL_MS = 5000;

const currencyFormatters = {
  USD: new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 4 }),
  EUR: new Intl.NumberFormat('en-US', { style: 'currency', currency: 'EUR', minimumFractionDigits: 2, maximumFractionDigits: 4 }),
};

function CostCell({ book }: { book: CatalogBook }) {
  const usageEntries = Object.entries(book.TokenUsage ?? {});

  if (usageEntries.length === 0) {
    return <Text size="sm" c="dimmed">—</Text>;
  }

  const breakdown = usageEntries
    .map(([model, usage]) => `${model}: ${usage.input_tokens.toLocaleString()} in / ${usage.output_tokens.toLocaleString()} out`)
    .join('\n');

  return (
    <Tooltip label={breakdown} multiline maw={320} style={{ whiteSpace: 'pre-line' }}>
      <Stack gap={0}>
        <Text size="sm">{currencyFormatters.USD.format(book.CostUSD)}</Text>
        {book.CostEUR !== undefined && (
          <Text size="xs" c="dimmed">{currencyFormatters.EUR.format(book.CostEUR)}</Text>
        )}
      </Stack>
    </Tooltip>
  );
}

function ProgressStages({ book }: { book: CatalogBook }) {
  const failedKey = book.Failed ? FAILED_STAGE_TO_PROGRESS_KEY[book.FailedStage] : undefined;

  return (
    <Group gap={4} wrap="nowrap">
      {STAGES.map((stage) => {
        const failed = stage.key === failedKey;
        return (
          <Tooltip key={stage.key} label={failed ? book.ErrorMessage : stage.label} multiline maw={320}>
            <Box
              w={10}
              h={10}
              style={{
                borderRadius: '50%',
                backgroundColor: failed
                  ? 'var(--mantine-color-red-6)'
                  : book[stage.key]
                    ? 'var(--mantine-color-teal-6)'
                    : 'var(--mantine-color-gray-6)',
              }}
            />
          </Tooltip>
        );
      })}
    </Group>
  );
}

export default function CataloguePage() {
  const [books, setBooks] = useState<CatalogBook[] | null>(null);
  const [error, setError] = useState('');
  const [retryingID, setRetryingID] = useState<string | null>(null);
  const [retryError, setRetryError] = useState('');

  function refresh() {
    getCatalog()
      .then(setBooks)
      .catch((cause) => setError(cause instanceof Error ? cause.message : 'Unable to load the catalogue.'));
  }

  useEffect(() => {
    refresh();
    const interval = setInterval(refresh, POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, []);

  async function handleRetry(bookID: string) {
    setRetryingID(bookID);
    setRetryError('');
    try {
      await retryBook(bookID);
      refresh();
    } catch (cause) {
      setRetryError(cause instanceof Error ? cause.message : 'Retry failed.');
    } finally {
      setRetryingID(null);
    }
  }

  return (
    <Stack maw={900} mx="auto" mt={24} gap="lg">
      <Group justify="space-between">
        <Title order={1}>Catalogue</Title>
        <ActionIcon variant="subtle" onClick={refresh} aria-label="Refresh">↻</ActionIcon>
      </Group>

      {error && <Alert color="red">{error}</Alert>}
      {retryError && <Alert color="red" onClose={() => setRetryError('')} withCloseButton>{retryError}</Alert>}

      {!books && !error && <Group justify="center" mt={48}><Loader /></Group>}

      {books && books.length === 0 && <Text c="dimmed">No books in the catalogue yet.</Text>}

      {books && books.length > 0 && (
        <Card withBorder padding={0}>
          <Table verticalSpacing="sm" horizontalSpacing="md">
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Book</Table.Th>
                <Table.Th>Language</Table.Th>
                <Table.Th>Progress</Table.Th>
                <Table.Th>Cost</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {books.map((book) => (
                <Table.Tr key={book.ID}>
                  <Table.Td>
                    <Group wrap="nowrap" gap="sm">
                      <div style={{ width: 32, flexShrink: 0 }}>
                        <BookCover title={book.Title} coverURL={book.CoverURL} height={48} />
                      </div>
                      <Stack gap={0}>
                        <Text size="sm" fw={600} lineClamp={1}>{book.Title}</Text>
                        <Text size="xs" c="dimmed" lineClamp={1}>{(book.AuthorNames ?? []).join(', ')}</Text>
                      </Stack>
                    </Group>
                  </Table.Td>
                  <Table.Td>
                    <Badge variant="light">{book.Language}</Badge>
                  </Table.Td>
                  <Table.Td>
                    <ProgressStages book={book} />
                  </Table.Td>
                  <Table.Td>
                    <CostCell book={book} />
                  </Table.Td>
                  <Table.Td>
                    {book.Failed && (
                      <Button
                        size="xs"
                        color="red"
                        variant="light"
                        loading={retryingID === book.ID}
                        onClick={() => handleRetry(book.ID)}
                      >
                        Retry
                      </Button>
                    )}
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Card>
      )}
    </Stack>
  );
}
