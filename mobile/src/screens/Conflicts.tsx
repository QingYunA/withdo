import { Text, View } from 'react-native';
import { appStore } from '../data';
import type { EntityData } from '../model';
import { dateLabel } from '../i18n';
import {
  Button,
  Empty,
  ErrorMessage,
  Label,
  Page,
  Section,
  useAction,
} from '../ui/components';
import { type, useUI } from '../ui/theme';

function Version({
  label,
  data,
  onKeep,
  actionLabel,
  disabled,
}: {
  label: string;
  data: EntityData | null;
  onKeep: () => void;
  actionLabel: string;
  disabled: boolean;
}) {
  const { t, colors, locale, snapshot } = useUI();
  return (
    <View
      style={{
        gap: 12,
        paddingVertical: 16,
        borderBottomWidth: 1,
        borderBottomColor: colors.border,
      }}
    >
      <Text style={[type.label, { color: colors.secondary }]}>{label}</Text>
      {data ? (
        <>
          <Label style={{ fontWeight: '600' }}>
            {'title' in data
              ? data.title
              : 'name' in data
                ? data.name
                : data.summary}
          </Label>
          {'notes' in data && data.notes ? <Label>{data.notes}</Label> : null}
          {'status' in data && data.status ? (
            <Label secondary>{t(`status_${data.status}`)}</Label>
          ) : null}
          {'projectId' in data ? (
            <Label secondary>
              {t('project')}:{' '}
              {snapshot.projects.find(
                (project) => project.id === data.projectId,
              )?.name ?? t('noProject')}
            </Label>
          ) : null}
          {'parentId' in data && data.parentId ? (
            <Label secondary>
              {t('parentTask')}:{' '}
              {snapshot.tasks.find((task) => task.id === data.parentId)
                ?.title ?? t('taskNotFound')}
            </Label>
          ) : null}
          {'priority' in data ? (
            <Label secondary>
              {t('priority')}: {data.priority}
            </Label>
          ) : null}
          {'dueAt' in data && data.dueAt ? (
            <Label secondary>
              {t('due')}: {dateLabel(data.dueAt, locale, true)}
            </Label>
          ) : null}
          {'remindAt' in data && data.remindAt ? (
            <Label secondary>
              {t('reminder')}: {dateLabel(data.remindAt, locale, true)}
            </Label>
          ) : null}
          {'archivedAt' in data && data.archivedAt ? (
            <Label secondary>{t('archived')}</Label>
          ) : null}
        </>
      ) : (
        <Label>{t('deletedVersion')}</Label>
      )}
      <Button
        label={actionLabel}
        kind="tonal"
        disabled={disabled}
        onPress={onKeep}
      />
    </View>
  );
}
export function ConflictsScreen() {
  const { snapshot, t } = useUI();
  const action = useAction();
  return (
    <Page title={t('conflicts')} back>
      {!snapshot.conflicts.length ? (
        <Empty
          title={t('noConflicts')}
          body={t('noConflictsBody')}
          icon="check"
        />
      ) : (
        <>
          <Label secondary>{t('conflictsHelp')}</Label>
          <ErrorMessage message={action.error} />
          {snapshot.conflicts.map((conflict) => (
            <Section
              key={conflict.mutation.id}
              title={
                conflict.mutation.type === 'project'
                  ? t('project')
                  : conflict.mutation.type === 'task'
                    ? t('task')
                    : t('latestReport')
              }
            >
              <Version
                label={t('localVersion')}
                data={conflict.mutation.data}
                disabled={action.busy}
                actionLabel={t('keepLocal')}
                onKeep={() =>
                  void action.run(() =>
                    appStore.resolveConflict(conflict.mutation.id, 'local'),
                  )
                }
              />
              <Version
                label={t('remoteVersion')}
                data={
                  conflict.remote?.deleted
                    ? null
                    : (conflict.remote?.data ?? null)
                }
                disabled={action.busy}
                actionLabel={t('keepRemote')}
                onKeep={() =>
                  void action.run(() =>
                    appStore.resolveConflict(conflict.mutation.id, 'remote'),
                  )
                }
              />
            </Section>
          ))}
        </>
      )}
    </Page>
  );
}
