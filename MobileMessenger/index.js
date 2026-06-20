/**
 * @format
 */

import {AppRegistry} from 'react-native';
import App from './App';
import {name as appName} from './app.json';

// Ensure the App is registered and Background Sync can be bootstrapped from anywhere
AppRegistry.registerComponent(appName, () => App);
