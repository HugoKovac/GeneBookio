import { Stack, Text, Title } from '@mantine/core';

export default function AboutPage() {
  return (
    <Stack px="lg" pt="lg" gap="sm">
      <Title order={2} style={{ fontSize: 28 }}>About</Title>
      <Text c="dimmed">GeneBookio turns your books into narrated audio.</Text>
    </Stack>
  );
}
