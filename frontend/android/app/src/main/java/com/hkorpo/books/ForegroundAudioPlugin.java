package com.hkorpo.books;

import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

// Bridges PlayerContext (frontend/src/features/player) to
// PlaybackForegroundService so background audio playback survives Doze /
// app-standby restrictions on Android. @capgo/capacitor-native-audio handles
// the audio itself and the lock-screen "Now Playing" notification, but
// deliberately doesn't manage a foreground service — see its
// `backgroundPlayback` option in node_modules/@capgo/capacitor-native-audio.
@CapacitorPlugin(name = "ForegroundAudio")
public class ForegroundAudioPlugin extends Plugin {

  @PluginMethod
  public void start(PluginCall call) {
    String title = call.getString("title", "");
    PlaybackForegroundService.start(getContext(), title);
    call.resolve();
  }

  @PluginMethod
  public void stop(PluginCall call) {
    PlaybackForegroundService.stop(getContext());
    call.resolve();
  }
}
