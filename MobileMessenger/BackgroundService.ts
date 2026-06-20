import BackgroundTimer from 'react-native-background-timer';

export const startBackgroundSync = (syncCallback: () => void) => {
  console.log('[BackgroundService] Starting background execution task...');

  // Start a timer that runs continuously even in background
  BackgroundTimer.runBackgroundTimer(() => {
      console.log('[BackgroundService] Heartbeat triggering sync logic...');
      syncCallback();
  },
  15000); // Trigger every 15 seconds for testing/sync logic
};

export const stopBackgroundSync = () => {
  console.log('[BackgroundService] Stopping background execution task.');
  BackgroundTimer.stopBackgroundTimer();
};
