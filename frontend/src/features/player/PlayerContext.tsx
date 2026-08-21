import { useMemo, useRef, useState, type ReactNode } from 'react';
import { PlayerContext as Context, type PlayerContextValue, type PlayerTrack } from './context';

export function PlayerProvider({ children }: { children: ReactNode }) {
  const audioRef = useRef<HTMLAudioElement>(null);

  const [track, setTrack] = useState<PlayerTrack | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isExpanded, setIsExpanded] = useState(false);
  const [isBuffering, setIsBuffering] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [bufferedEnd, setBufferedEnd] = useState(0);
  const [error, setError] = useState(false);

  const updateBuffered = () => {
    const audio = audioRef.current;
    if (!audio || audio.buffered.length === 0) return;
    setBufferedEnd(audio.buffered.end(audio.buffered.length - 1));
  };

  const value = useMemo<PlayerContextValue>(() => {
    const seek = (time: number) => {
      const audio = audioRef.current;
      if (!audio) return;
      audio.currentTime = time;
      setCurrentTime(time);
    };

    const skip = (deltaSeconds: number) => {
      const audio = audioRef.current;
      if (!audio) return;
      const target = Math.min(Math.max(audio.currentTime + deltaSeconds, 0), duration || audio.duration || 0);
      seek(target);
    };

    const playTrack = (next: PlayerTrack) => {
      const audio = audioRef.current;
      if (!audio) return;

      if (track?.bookID === next.bookID) {
        void audio.play();
        return;
      }

      setTrack(next);
      setError(false);
      setCurrentTime(0);
      setDuration(0);
      setBufferedEnd(0);
      audio.src = next.src;
      void audio.play();
    };

    const togglePlay = () => {
      const audio = audioRef.current;
      if (!audio) return;
      if (audio.paused) void audio.play();
      else audio.pause();
    };

    return {
      track,
      isPlaying,
      isExpanded,
      isBuffering,
      currentTime,
      duration,
      bufferedEnd,
      error,
      playTrack,
      togglePlay,
      seek,
      skip,
      expand: () => setIsExpanded(true),
      minimize: () => setIsExpanded(false),
    };
  }, [track, isPlaying, isExpanded, isBuffering, currentTime, duration, bufferedEnd, error]);

  return (
    <Context.Provider value={value}>
      {children}
      <audio
        ref={audioRef}
        preload="metadata"
        onPlay={() => setIsPlaying(true)}
        onPause={() => setIsPlaying(false)}
        onWaiting={() => setIsBuffering(true)}
        onCanPlay={() => setIsBuffering(false)}
        onLoadedMetadata={(e) => setDuration(e.currentTarget.duration)}
        onDurationChange={(e) => setDuration(e.currentTarget.duration)}
        onTimeUpdate={(e) => setCurrentTime(e.currentTarget.currentTime)}
        onProgress={updateBuffered}
        onError={() => setError(true)}
        onEnded={() => setIsPlaying(false)}
        style={{ display: 'none' }}
      />
    </Context.Provider>
  );
}
