import { registerPlugin } from '@capacitor/core';

// Bridges to android/app/src/main/java/com/hkorpo/books/ForegroundAudioPlugin.java,
// a local (non-npm) plugin with no web/iOS implementation — iOS keeps
// background audio alive via the UIBackgroundModes "audio" entry in
// Info.plist instead, so this must only be called on Android (see PlayerContext).
interface ForegroundAudioPlugin {
  start(options: { title: string }): Promise<void>;
  stop(): Promise<void>;
}

export const ForegroundAudio = registerPlugin<ForegroundAudioPlugin>('ForegroundAudio');
