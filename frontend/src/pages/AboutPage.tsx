import { Stack, Text, Title } from '@mantine/core';
import { useTranslation } from 'react-i18next';

export default function AboutPage() {
  const { t } = useTranslation();

  return (
    <Stack px="lg" pt="lg" gap="sm">
      <Title order={2} style={{ fontSize: 28 }}>{t('about.title')}</Title>
      <Text c="dimmed">{t('about.description')}</Text>
    </Stack>
  );
}
