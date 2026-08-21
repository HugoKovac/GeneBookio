import { Capacitor } from '@capacitor/core';
import { NativeAudio } from '@capgo/capacitor-native-audio';
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { PlayerContext as Context, type PlayerContextValue, type PlayerTrack } from './context';
import { ForegroundAudio } from './foregroundAudio';

const isNative = Capacitor.isNativePlatform();
// iOS keeps playing in the background via the UIBackgroundModes "audio" entry
// in Info.plist alone; Android needs its own foreground service (no such
// plugin/web implementation exists for iOS — see foregroundAudio.ts).
const isAndroid = Capacitor.getPlatform() === 'android';

// NativeAudio only ever holds one asset at a time for us — a single fixed id
// keeps preload/play/stop calls from having to track per-book asset ids.
const NATIVE_ASSET_ID = 'book-track';

// On native (iOS/Android) playback goes through @capgo/capacitor-native-audio
// instead of an <audio> element so it keeps playing — with lock-screen /
// Control Center transport controls — once the app is backgrounded or the
// screen locks, which a WebView <audio> element can't do on its own. Web
// keeps the plain <audio> element below.
export function PlayerProvider({ children }: { children: ReactNode }) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const nativeLoadedBookID = useRef<string | null>(null);

  const [track, setTrack] = useState<PlayerTrack | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isExpanded, setIsExpanded] = useState(false);
  const [isBuffering, setIsBuffering] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [bufferedEnd, setBufferedEnd] = useState(0);
  const [playbackRate, setPlaybackRateState] = useState(1);
  const [error, setError] = useState(false);

  const updateBuffered = () => {
    const audio = audioRef.current;
    if (!audio || audio.buffered.length === 0) return;
    setBufferedEnd(audio.buffered.end(audio.buffered.length - 1));
  };

  useEffect(() => {
    if (!isNative) return;

    void NativeAudio.configure({
      focus: true,
      background: true,
      showNotification: true,
      backgroundPlayback: true,
    });

    const handles = [
      NativeAudio.addListener('playbackState', (event) => {
        if (event.assetId !== NATIVE_ASSET_ID) return;
        setIsPlaying(event.isPlaying);
        if (typeof event.currentTime === 'number') setCurrentTime(event.currentTime);
        if (typeof event.duration === 'number' && event.duration > 0) setDuration(event.duration);
      }),
      NativeAudio.addListener('currentTime', (event) => {
        if (event.assetId !== NATIVE_ASSET_ID) return;
        setCurrentTime(event.currentTime);
      }),
      NativeAudio.addListener('complete', (event) => {
        if (event.assetId !== NATIVE_ASSET_ID) return;
        setIsPlaying(false);
      }),
    ];

    return () => {
      void Promise.all(handles).then((resolved) => resolved.forEach((handle) => handle.remove()));
    };
  }, []);

  useEffect(() => {
    if (!isAndroid) return;
    if (isPlaying && track) {
      void ForegroundAudio.start({ title: track.title });
    } else {
      void ForegroundAudio.stop();
    }
  }, [isPlaying, track]);

  const value = useMemo<PlayerContextValue>(() => {
    const seek = (time: number) => {
      if (isNative) {
        if (!track) return;
        void NativeAudio.setCurrentTime({ assetId: NATIVE_ASSET_ID, time });
        setCurrentTime(time);
        return;
      }

      const audio = audioRef.current;
      if (!audio) return;
      audio.currentTime = time;
      setCurrentTime(time);
    };

    const skip = (deltaSeconds: number) => {
      const base = isNative ? currentTime : (audioRef.current?.currentTime ?? currentTime);
      const target = Math.min(Math.max(base + deltaSeconds, 0), duration || 0);
      seek(target);
    };

    const playTrackNative = async (next: PlayerTrack) => {
      setError(false);

      if (track?.bookID === next.bookID) {
        try {
          await NativeAudio.resume({ assetId: NATIVE_ASSET_ID });
        } catch {
          setError(true);
        }
        return;
      }

      if (nativeLoadedBookID.current) {
        try {
          await NativeAudio.stop({ assetId: NATIVE_ASSET_ID });
          await NativeAudio.unload({ assetId: NATIVE_ASSET_ID });
        } catch {
          // best-effort cleanup of the previous asset
        }
      }

      setTrack(next);
      setCurrentTime(0);
      setDuration(0);
      setIsBuffering(true);

      try {
        await NativeAudio.preload({
          assetId: NATIVE_ASSET_ID,
          assetPath: next.src,
          isUrl: true,
          notificationMetadata: {
            title: next.title,
            artist: next.author,
            artworkUrl: next.coverURL || undefined,
          },
        });
        nativeLoadedBookID.current = next.bookID;
        await NativeAudio.play({ assetId: NATIVE_ASSET_ID });
        setIsBuffering(false);
        try {
          await NativeAudio.setRate({ assetId: NATIVE_ASSET_ID, rate: playbackRate });
        } catch {
          // non-fatal — playback continues at the default rate
        }

        try {
          const { duration: loadedDuration } = await NativeAudio.getDuration({ assetId: NATIVE_ASSET_ID });
          if (loadedDuration > 0) setDuration(loadedDuration);
        } catch {
          // duration will still arrive via the playbackState/currentTime listeners
        }
      } catch {
        setError(true);
        setIsBuffering(false);
      }
    };

    const playTrack = (next: PlayerTrack) => {
      if (isNative) {
        void playTrackNative(next);
        return;
      }

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
      audio.playbackRate = playbackRate;
      void audio.play();
    };

    const setPlaybackRate = (rate: number) => {
      setPlaybackRateState(rate);
      if (isNative) {
        if (track) void NativeAudio.setRate({ assetId: NATIVE_ASSET_ID, rate });
        return;
      }

      const audio = audioRef.current;
      if (audio) audio.playbackRate = rate;
    };

    const togglePlay = () => {
      if (isNative) {
        if (!track) return;
        void (isPlaying
          ? NativeAudio.pause({ assetId: NATIVE_ASSET_ID })
          : NativeAudio.resume({ assetId: NATIVE_ASSET_ID }));
        return;
      }

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
      playbackRate,
      error,
      playTrack,
      togglePlay,
      seek,
      skip,
      setPlaybackRate,
      expand: () => setIsExpanded(true),
      minimize: () => setIsExpanded(false),
    };
  }, [track, isPlaying, isExpanded, isBuffering, currentTime, duration, bufferedEnd, playbackRate, error]);

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
