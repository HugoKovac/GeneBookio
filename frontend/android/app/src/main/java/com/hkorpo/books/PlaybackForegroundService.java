package com.hkorpo.books;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.Service;
import android.content.Context;
import android.content.Intent;
import android.content.pm.ServiceInfo;
import android.os.Build;
import android.os.IBinder;
import androidx.core.app.NotificationCompat;

// Keeps the app process alive as an Android foreground service while a book
// is playing in the background / with the screen locked. @capgo/capacitor-native-audio
// (see ForegroundAudioPlugin) posts its own richer "Now Playing" notification
// with real transport controls for the audio itself — this service only
// exists to satisfy Android's requirement that background media playback run
// inside a foreground service, so it uses its own separate, low-priority
// notification rather than depending on the plugin's private notification id.
public class PlaybackForegroundService extends Service {

  private static final String CHANNEL_ID = "book_playback_service";
  private static final int NOTIFICATION_ID = 4200;

  public static final String EXTRA_TITLE = "title";

  @Override
  public int onStartCommand(Intent intent, int flags, int startId) {
    String title = intent != null ? intent.getStringExtra(EXTRA_TITLE) : null;
    startForeground(NOTIFICATION_ID, buildNotification(title), foregroundServiceType());
    return START_STICKY;
  }

  private int foregroundServiceType() {
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
      return ServiceInfo.FOREGROUND_SERVICE_TYPE_MEDIA_PLAYBACK;
    }
    return 0;
  }

  private Notification buildNotification(String title) {
    createChannelIfNeeded();

    PendingIntent contentIntent = PendingIntent.getActivity(
      this,
      0,
      new Intent(this, MainActivity.class),
      Build.VERSION.SDK_INT >= Build.VERSION_CODES.M ? PendingIntent.FLAG_IMMUTABLE : 0
    );

    return new NotificationCompat.Builder(this, CHANNEL_ID)
      .setContentTitle(getString(R.string.app_name))
      .setContentText(title != null ? title : "Playing audio")
      .setSmallIcon(R.mipmap.ic_launcher)
      .setContentIntent(contentIntent)
      .setOngoing(true)
      .setPriority(NotificationCompat.PRIORITY_LOW)
      .build();
  }

  private void createChannelIfNeeded() {
    if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return;
    NotificationManager manager = getSystemService(NotificationManager.class);
    if (manager.getNotificationChannel(CHANNEL_ID) != null) return;
    NotificationChannel channel = new NotificationChannel(
      CHANNEL_ID,
      "Background playback",
      NotificationManager.IMPORTANCE_LOW
    );
    channel.setDescription("Keeps audio playing while the app is in the background");
    manager.createNotificationChannel(channel);
  }

  @Override
  public IBinder onBind(Intent intent) {
    return null;
  }

  public static void start(Context context, String title) {
    Intent intent = new Intent(context, PlaybackForegroundService.class);
    intent.putExtra(EXTRA_TITLE, title);
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
      context.startForegroundService(intent);
    } else {
      context.startService(intent);
    }
  }

  public static void stop(Context context) {
    context.stopService(new Intent(context, PlaybackForegroundService.class));
  }
}
