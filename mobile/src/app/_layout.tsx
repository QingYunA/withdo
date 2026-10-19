import { useCallback, useEffect, useState } from 'react';
import { ActivityIndicator, View } from 'react-native';
import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { appStore } from '../data';
import { Button, Empty, ErrorMessage } from '../ui/components';
import { UIProvider, useUI } from '../ui/theme';

function Root() {
  const { colors, t, isDark, snapshot } = useUI();
  const [error, setError] = useState<string | null>(null);
  const initialize = useCallback(() => {
    setError(null);
    void appStore
      .initialize()
      .catch((e) => setError(e instanceof Error ? e.message : String(e)));
  }, []);
  useEffect(() => {
    void appStore
      .initialize()
      .catch((e) => setError(e instanceof Error ? e.message : String(e)));
  }, []);
  const initializationError = error ?? snapshot.syncError;
  return (
    <View style={{ flex: 1, backgroundColor: colors.surface }}>
      <StatusBar style={isDark ? 'light' : 'dark'} />
      {!snapshot.ready ? (
        initializationError ? (
          <Empty
            title={t('openingFailed')}
            action={
              <>
                <ErrorMessage message={initializationError} />
                <Button label={t('retry')} onPress={initialize} />
              </>
            }
          />
        ) : (
          <Empty
            title={t('loading')}
            action={<ActivityIndicator color={colors.primary} />}
          />
        )
      ) : (
        <Stack
          screenOptions={{
            headerShown: false,
            contentStyle: { backgroundColor: colors.surface },
          }}
        />
      )}
    </View>
  );
}
export default function RootLayout() {
  return (
    <SafeAreaProvider>
      <UIProvider>
        <Root />
      </UIProvider>
    </SafeAreaProvider>
  );
}
