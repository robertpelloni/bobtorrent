jest.mock('react-native-background-timer', () => {
  return {
    runBackgroundTimer: jest.fn(),
    stopBackgroundTimer: jest.fn(),
  };
});
