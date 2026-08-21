import { createContext } from 'react';

export type PlayerTrack = {
  bookID: string;
  title: string;
  author: string;
  coverURL: string;
  src: string;
};

export type PlayerContextValue = {
  track: PlayerTrack | null;
  isPlaying: boolean;
  isExpanded: boolean;
  isBuffering: boolean;
  currentTime: number;
  duration: number;
  bufferedEnd: number;
  error: boolean;
  playTrack: (track: PlayerTrack) => void;
  togglePlay: () => void;
  seek: (time: number) => void;
  skip: (deltaSeconds: number) => void;
  expand: () => void;
  minimize: () => void;
};

export const PlayerContext = createContext<PlayerContextValue | null>(null);
