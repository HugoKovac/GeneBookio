import React from 'react';
import ReactDOM from 'react-dom/client';
import { MantineProvider } from '@mantine/core';
import '@mantine/core/styles.css';
import './i18n';
import App from './App';
import { AuthProvider } from './features/auth/AuthContext';
import { PlayerProvider } from './features/player/PlayerContext';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <MantineProvider defaultColorScheme="dark">
      <AuthProvider>
        <PlayerProvider><App /></PlayerProvider>
      </AuthProvider>
    </MantineProvider>
  </React.StrictMode>
);
