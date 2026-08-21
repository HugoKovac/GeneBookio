import type { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.hkorpo.books',
  appName: 'Books',
  webDir: 'dist',
  server: {
    androidScheme: 'http',
  },
};

export default config;
