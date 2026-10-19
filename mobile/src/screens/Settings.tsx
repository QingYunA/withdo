import { useState } from 'react';
import { Alert, Linking, Switch, View } from 'react-native';
import { router } from 'expo-router';
import { appStore } from '../data';
import { dateLabel } from '../i18n';
import type { Appearance, Language } from '../model';
import {
  Button,
  Choice,
  ErrorMessage,
  Field,
  Label,
  Page,
  Row,
  Section,
  useAction,
} from '../ui/components';
import { useUI } from '../ui/theme';

export function SettingsScreen() {
  const { snapshot, colors, t, locale } = useUI();
  const [serverUrl, setServerUrl] = useState(snapshot.settings.serverUrl);
  const [token, setToken] = useState('');
  const action = useAction();
  const connection = useAction();
  const connectionChanged =
    serverUrl.trim() !== snapshot.settings.serverUrl || !!token;
  const connect = () =>
    void connection.run(async () => {
      await appStore.updateSettings(
        { serverUrl: serverUrl.trim() },
        token || undefined,
      );
      setToken('');
      await appStore.sync();
    });
  const disconnect = () =>
    Alert.alert(t('disconnectQuestion'), t('disconnectBody'), [
      { text: t('cancel'), style: 'cancel' },
      {
        text: t('disconnect'),
        onPress: () =>
          void connection.run(async () => {
            await appStore.updateSettings({ serverUrl: '' }, '');
            setServerUrl('');
            setToken('');
          }),
      },
    ]);
  const state =
    snapshot.syncState === 'syncing'
      ? t('syncing')
      : snapshot.syncState === 'offline'
        ? t('offline')
        : snapshot.syncState === 'error'
          ? t('syncError')
          : snapshot.settings.serverUrl
            ? t('connected')
            : t('notConnected');
  return (
    <Page title={t('settings')}>
      <Section title={t('language')}>
        <Choice<Language>
          label={t('language')}
          selected={snapshot.settings.language}
          onSelect={(language) =>
            void action.run(() => appStore.updateSettings({ language }))
          }
          options={[
            { value: 'system', label: t('system') },
            { value: 'zh', label: '中文' },
            { value: 'en', label: 'English' },
          ]}
        />
      </Section>
      <Section title={t('appearance')}>
        <Choice<Appearance>
          label={t('appearance')}
          selected={snapshot.settings.appearance}
          onSelect={(appearance) =>
            void action.run(() => appStore.updateSettings({ appearance }))
          }
          options={[
            { value: 'system', label: t('system') },
            { value: 'light', label: t('light') },
            { value: 'dark', label: t('dark') },
          ]}
        />
      </Section>
      <Section title={t('reminders')}>
        <Row>
          <View style={{ flex: 1, gap: 4 }}>
            <Label>{t('reminders')}</Label>
            <Label secondary>{t('remindersHelp')}</Label>
          </View>
          <Switch
            accessibilityLabel={t('reminders')}
            value={snapshot.settings.remindersEnabled}
            onValueChange={(remindersEnabled) =>
              void action.run(() =>
                appStore.updateSettings({ remindersEnabled }),
              )
            }
            disabled={action.busy}
            trackColor={{ false: colors.border, true: colors.primary }}
          />
        </Row>
      </Section>
      <ErrorMessage message={snapshot.reminderError} />
      {snapshot.reminderError && (
        <Button
          label={t('openSystemSettings')}
          kind="text"
          onPress={() => void action.run(() => Linking.openSettings())}
        />
      )}
      <ErrorMessage message={action.error} />
      <Section title={t('sync')}>
        <Label
          style={{
            color: snapshot.syncState === 'error' ? colors.error : colors.text,
          }}
        >
          {state}
        </Label>
        <Label secondary>
          {t('lastSynced')}:{' '}
          {snapshot.lastSyncedAt
            ? dateLabel(snapshot.lastSyncedAt, locale, true)
            : t('neverSynced')}
        </Label>
        {!!snapshot.pendingCount && (
          <Label secondary>
            {snapshot.pendingCount} {t('changes')} · {t('pending')}
          </Label>
        )}
        {!!snapshot.conflicts.length && (
          <Button
            label={`${t('conflicts')} · ${snapshot.conflicts.length}`}
            kind="tonal"
            onPress={() => router.push('/conflicts')}
          />
        )}
        <ErrorMessage message={snapshot.syncError} />
        {snapshot.settings.serverUrl ? (
          <Button
            label={t('syncNow')}
            kind="tonal"
            busy={snapshot.syncState === 'syncing'}
            disabled={connection.busy}
            onPress={() => void connection.run(() => appStore.sync())}
          />
        ) : null}
      </Section>
      <Section title={t('connection')}>
        <Label secondary>{t('connectionHelp')}</Label>
        <Field
          label={t('serverUrl')}
          placeholder="https://sync.example.com"
          autoCapitalize="none"
          autoCorrect={false}
          keyboardType="url"
          value={serverUrl}
          onChangeText={setServerUrl}
        />
        <Field
          label={t('accessKey')}
          placeholder={t('keyPlaceholder')}
          autoCapitalize="none"
          autoCorrect={false}
          secureTextEntry
          value={token}
          onChangeText={setToken}
        />
        <ErrorMessage message={connection.error} />
        <Button
          label={t('connect')}
          busy={connection.busy}
          disabled={!serverUrl.trim() || !connectionChanged}
          onPress={connect}
        />
        {snapshot.settings.serverUrl ? (
          <Button
            label={t('disconnect')}
            kind="text"
            disabled={connection.busy}
            onPress={disconnect}
          />
        ) : null}
      </Section>
    </Page>
  );
}
