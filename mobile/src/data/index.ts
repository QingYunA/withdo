import { useEffect, useSyncExternalStore } from 'react';
import { AppState } from 'react-native';
import * as SQLite from 'expo-sqlite';
import * as SecureStore from 'expo-secure-store';
import * as Crypto from 'expo-crypto';
import * as Network from 'expo-network';
import type { AppStore } from '../model';
import { createAppStore } from './store';
import { reconcileReminders } from './reminders';

// Scope credentials to the endpoint so switching servers never sends an old token to a new host.
const tokenKey = (url: string) => `adhd-todo-token.${Array.from(url).map(char => char.codePointAt(0)!.toString(16)).join('-')}`;
const core = createAppStore({
  openDatabase: () => SQLite.openDatabaseAsync('adhd-todo.db'),
  uuid: () => Crypto.randomUUID(),
  getToken: url => SecureStore.getItemAsync(tokenKey(url)),
  setToken: async (url, token) => {
    if (token) await SecureStore.setItemAsync(tokenKey(url), token);
    else await SecureStore.deleteItemAsync(tokenKey(url));
  },
  reconcileReminders,
  autoSync: true,
});
let listenersInstalled = false;
export const appStore: AppStore = {
  ...core,
  async initialize() {
    await core.initialize();
    if (listenersInstalled) return;
    listenersInstalled = true;
    AppState.addEventListener('change', state => {
      if (state === 'active') void core.sync().catch(() => undefined);
    });
    Network.addNetworkStateListener(state => {
      if (AppState.currentState === 'active' && state.isConnected && state.isInternetReachable !== false) void core.sync().catch(() => undefined);
    });
  },
};

export function useAppSnapshot() {
  useEffect(() => { void appStore.initialize().catch(() => undefined); }, []);
  return useSyncExternalStore(appStore.subscribe, appStore.getSnapshot, appStore.getSnapshot);
}
