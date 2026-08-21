import { createContext } from 'react';

export type PlayerTrack = {
  bookID: string;
  title: string;
  author: string;
  coverURL: string;
  src: string;
};

// Playback speed presets offered in NowPlayingOverlay's speed control.
export const PLAYBACK_RATES: number[] = [1, 1.5, 1.8, 2];

export type PlayerContextValue = {
  track: PlayerTrack | null;
  isPlaying: boolean;
  isExpanded: boolean;
  isBuffering: boolean;
  currentTime: number;
  duration: number;
  bufferedEnd: number;
  playbackRate: number;
  error: boolean;
  playTrack: (track: PlayerTrack) => void;
  togglePlay: () => void;
  seek: (time: number) => void;
  skip: (deltaSeconds: number) => void;
  setPlaybackRate: (rate: number) => void;
  expand: () => void;
  minimize: () => void;
};

export const PlayerContext = createContext<PlayerContextValue | null>(null);
