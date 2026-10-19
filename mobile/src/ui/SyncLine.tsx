import { Pressable, Text } from 'react-native';
import { router } from 'expo-router';
import { type, useUI } from './theme';
export function SyncLine() {
  const { snapshot, colors, t } = useUI();
  const label =
    snapshot.syncState === 'syncing'
      ? t('syncing')
      : snapshot.syncState === 'offline'
        ? t('offline')
        : snapshot.syncState === 'error'
          ? t('syncError')
          : snapshot.conflicts.length
            ? `${t('conflicts')} · ${snapshot.conflicts.length}`
            : snapshot.pendingCount && snapshot.settings.serverUrl
              ? `${snapshot.pendingCount} ${t('changes')} · ${t('pending')}`
              : t('saved');
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`${label}. ${t('settings')}`}
      onPress={() => router.push('/settings')}
      style={{ minHeight: 48, justifyContent: 'center' }}
    >
      <Text
        style={[
          type.label,
          {
            color:
              snapshot.syncState === 'error' ? colors.error : colors.secondary,
          },
        ]}
      >
        {label}
      </Text>
    </Pressable>
  );
}
