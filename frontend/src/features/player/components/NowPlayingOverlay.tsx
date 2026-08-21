import { ActionIcon, Alert, Box, Group, Image, Loader, Slider, Stack, Text, Title, UnstyledButton } from '@mantine/core';
import { IconChevronDown, IconPlayerPauseFilled, IconPlayerPlayFilled, IconRewindBackward15, IconRewindForward15 } from '@tabler/icons-react';
import { useState } from 'react';
import { usePlayer } from '../usePlayer';

function formatTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return '0:00';
  const minutes = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${minutes}:${secs.toString().padStart(2, '0')}`;
}

// NowPlayingOverlay is the full-screen "now playing" view, opened from
// MiniPlayerBar: the current book's cover blurred into a background, large
// artwork, a draggable progress bar, and playback controls.
export default function NowPlayingOverlay() {
  const {
    track, isExpanded, isPlaying, isBuffering, currentTime, duration, error,
    togglePlay, seek, skip, minimize,
  } = usePlayer();
  const [dragValue, setDragValue] = useState<number | null>(null);

  if (!track || !isExpanded) return null;

  const displayTime = dragValue ?? currentTime;

  return (
    <Box style={{ position: 'fixed', inset: 0, zIndex: 300, overflow: 'hidden' }}>
      {track.coverURL && (
        <Box
          style={{
            position: 'absolute',
            inset: -40,
            backgroundImage: `url(${track.coverURL})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
            filter: 'blur(60px) saturate(1.4) brightness(0.55)',
            transform: 'scale(1.1)',
          }}
        />
      )}
      <Box style={{ position: 'absolute', inset: 0, background: 'var(--mantine-color-dark-7)', opacity: track.coverURL ? 0.55 : 1 }} />
      <Box style={{ position: 'absolute', inset: 0, background: 'linear-gradient(to bottom, rgba(0,0,0,0.25), rgba(0,0,0,0.8))' }} />

      <Stack
        style={{ position: 'relative', height: '100%' }}
        px="lg"
        pt="calc(env(safe-area-inset-top, 0px) + 20px)"
        pb="calc(env(safe-area-inset-bottom, 0px) + 24px)"
        justify="space-between"
      >
        <Group justify="center" style={{ position: 'relative' }}>
          <ActionIcon
            variant="transparent"
            c="white"
            size="lg"
            onClick={minimize}
            style={{ position: 'absolute', left: 0 }}
            aria-label="Minimize player"
          >
            <IconChevronDown size={26} />
          </ActionIcon>
          <Text c="gray.4" size="xs" fw={700} tt="uppercase" style={{ letterSpacing: 1 }}>Now Playing</Text>
        </Group>

        <Stack align="center" gap="xl">
          <Box
            w="100%"
            style={{
              maxWidth: 320,
              aspectRatio: '1',
              borderRadius: 16,
              overflow: 'hidden',
              boxShadow: '0 24px 60px rgba(0,0,0,0.5)',
            }}
          >
            <Image src={track.coverURL} h="100%" w="100%" fit="cover" fallbackSrc="" />
          </Box>
          <Stack gap={4} align="center">
            <Title order={3} c="white" ta="center">{track.title}</Title>
            <Text c="gray.4" ta="center">{track.author}</Text>
          </Stack>
        </Stack>

        <Stack gap="xl">
          {error ? (
            <Alert color="yellow" radius="lg">Audio unavailable for this book.</Alert>
          ) : (
            <>
              <Stack gap={4}>
                <Slider
                  value={displayTime}
                  max={duration || 0}
                  min={0}
                  step={0.1}
                  label={formatTime(displayTime)}
                  onChange={setDragValue}
                  onChangeEnd={(value) => {
                    seek(value);
                    setDragValue(null);
                  }}
                  disabled={!duration}
                  color="gray.0"
                  styles={{ track: { background: 'rgba(255,255,255,0.25)' } }}
                />
                <Group justify="space-between">
                  <Text size="xs" c="gray.4">{formatTime(displayTime)}</Text>
                  <Text size="xs" c="gray.4">{formatTime(duration)}</Text>
                </Group>
              </Stack>

              <Group justify="center" align="center" gap="xl">
                <ActionIcon variant="transparent" c="white" size="xl" onClick={() => skip(-15)} aria-label="Back 15 seconds">
                  <IconRewindBackward15 size={26} />
                </ActionIcon>

                <UnstyledButton
                  onClick={togglePlay}
                  disabled={isBuffering && !duration}
                  aria-label={isPlaying ? 'Pause' : 'Play'}
                  style={{
                    width: 68,
                    height: 68,
                    borderRadius: '50%',
                    background: 'white',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    boxShadow: '0 6px 16px rgba(0,0,0,0.35)',
                  }}
                >
                  {isBuffering && !duration ? (
                    <Loader size={24} color="dark" />
                  ) : isPlaying ? (
                    <IconPlayerPauseFilled size={28} color="black" />
                  ) : (
                    <IconPlayerPlayFilled size={28} color="black" />
                  )}
                </UnstyledButton>

                <ActionIcon variant="transparent" c="white" size="xl" onClick={() => skip(15)} aria-label="Forward 15 seconds">
                  <IconRewindForward15 size={26} />
                </ActionIcon>
              </Group>
            </>
          )}
        </Stack>
      </Stack>
    </Box>
  );
}
