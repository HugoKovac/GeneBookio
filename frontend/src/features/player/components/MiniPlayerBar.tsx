import { ActionIcon, Box, Group, Image, Loader, Text, UnstyledButton } from '@mantine/core';
import { IconPlayerPauseFilled, IconPlayerPlayFilled } from '@tabler/icons-react';
import { usePlayer } from '../usePlayer';

// MiniPlayerBar sits pinned above the bottom tab bar, Spotify-style, once a
// book is playing. Tapping it opens the full-screen NowPlayingOverlay;
// tapping the play/pause button toggles playback without expanding.
export default function MiniPlayerBar() {
  const { track, isPlaying, isBuffering, currentTime, duration, error, togglePlay, expand } = usePlayer();

  if (!track) return null;

  const progress = duration ? Math.min(100, (currentTime / duration) * 100) : 0;

  return (
    <Box
      style={{
        position: 'fixed',
        left: 0,
        right: 0,
        bottom: 'calc(64px + env(safe-area-inset-bottom, 0px))',
        zIndex: 200,
        background: 'var(--mantine-color-dark-6)',
        borderTop: '1px solid var(--mantine-color-dark-4)',
      }}
    >
      <Box style={{ height: 2, background: 'var(--mantine-color-dark-4)' }}>
        <Box
          style={{
            height: '100%',
            width: `${progress}%`,
            background: 'var(--mantine-primary-color-filled)',
            transition: 'width 0.2s linear',
          }}
        />
      </Box>

      <UnstyledButton onClick={expand} style={{ display: 'block', width: '100%' }} aria-label={`Open ${track.title}`}>
        <Group px="sm" py={6} gap="sm" wrap="nowrap">
          <Box w={40} h={40} style={{ flexShrink: 0, borderRadius: 6, overflow: 'hidden' }}>
            {track.coverURL && <Image src={track.coverURL} w={40} h={40} fit="cover" />}
          </Box>
          <Box style={{ flex: 1, minWidth: 0 }}>
            <Text size="sm" fw={600} truncate>{track.title}</Text>
            <Text size="xs" c={error ? 'yellow' : 'dimmed'} truncate>{error ? 'Audio unavailable' : track.author}</Text>
          </Box>
          <ActionIcon
            size="lg"
            radius="xl"
            disabled={error}
            onClick={(e) => {
              e.stopPropagation();
              togglePlay();
            }}
          >
            {isBuffering && !duration ? (
              <Loader size={16} color="white" />
            ) : isPlaying ? (
              <IconPlayerPauseFilled size={18} />
            ) : (
              <IconPlayerPlayFilled size={18} />
            )}
          </ActionIcon>
        </Group>
      </UnstyledButton>
    </Box>
  );
}
