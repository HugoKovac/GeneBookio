import { Center, Image, Text } from '@mantine/core';

export default function BookCover({ title, coverURL, height = 200 }: { title: string; coverURL: string; height?: number }) {
  if (coverURL) {
    return <Image src={coverURL} height={height} alt={title} fit="cover" radius="sm" />;
  }

  return (
    <Center h={height} bg="dark.5" style={{ borderRadius: 'var(--mantine-radius-sm)' }}>
      <Text size="sm" c="dimmed" ta="center" px="sm" lineClamp={3}>{title}</Text>
    </Center>
  );
}
