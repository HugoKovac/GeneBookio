import type { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.hkorpo.books',
  appName: 'Books',
  webDir: 'dist',
  server: {
    androidScheme: 'http',
  },
  plugins: {
    // Book audio is plain MP3/WAV from cmd/api's TTS output, never HLS —
    // skips bundling media3-exoplayer-hls (~4MB) for nothing.
    NativeAudio: {
      hls: false,
    },
  },
};

export default config;
