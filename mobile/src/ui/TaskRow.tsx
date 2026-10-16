import { Pressable, Text, View } from 'react-native';
import { router } from 'expo-router';
import type { Report, Task } from '../model';
import { appStore } from '../data';
import { dateLabel, statusKey } from '../i18n';
import { StatusIcon } from './icons';
import { type, useUI } from './theme';
import { ErrorMessage, useAction } from './components';

export function reportAge(
  iso: string,
  t: ReturnType<typeof useUI>['t'],
  now: number,
): string {
  const age = Math.max(0, now - new Date(iso).getTime());
  if (age < 60_000) return t('justNow');
  if (age < 3_600_000) return `${Math.floor(age / 60_000)} ${t('minutesAgo')}`;
  if (age < 86_400_000)
    return `${Math.floor(age / 3_600_000)} ${t('hoursAgo')}`;
  return `${Math.floor(age / 86_400_000)} ${t('daysAgo')}`;
}
export function latestReport(
  taskId: string,
  reports: Report[],
): Report | undefined {
  return reports
    .filter((r) => r.taskId === taskId)
    .sort((a, b) => b.createdAt.localeCompare(a.createdAt))[0];
}
export function TaskRow({
  task,
  agent = false,
}: {
  task: Task;
  agent?: boolean;
}) {
  const { colors, t, locale, snapshot, now } = useUI();
  const action = useAction();
  const openDetails =
    task.source === 'agent' || ['done', 'cancelled'].includes(task.status);
  const openTask = () =>
    router.push({ pathname: '/task/[id]', params: { id: task.id } });
  const project = snapshot.projects.find((p) => p.id === task.projectId);
  const report = agent ? latestReport(task.id, snapshot.reports) : undefined;
  const stale =
    report &&
    now - new Date(report.createdAt).getTime() > 86_400_000 &&
    !['done', 'cancelled'].includes(task.status);
  return (
    <View style={{ paddingHorizontal: 12 }}>
      <View style={{ flexDirection: 'row', alignItems: 'flex-start' }}>
        <Pressable
          accessibilityRole={openDetails ? 'button' : 'checkbox'}
          accessibilityLabel={`${t(openDetails ? 'openTask' : 'complete')}: ${task.title}`}
          accessibilityState={
            openDetails
              ? { disabled: action.busy }
              : { checked: false, disabled: action.busy }
          }
          disabled={action.busy}
          onPress={() => {
            if (openDetails) openTask();
            else
              void action.run(() =>
                appStore.updateTask(task.id, { status: 'done' }),
              );
          }}
          style={({ pressed }) => ({
            width: 48,
            minHeight: 64,
            paddingTop: 19,
            alignItems: 'center',
            opacity: pressed ? 0.6 : 1,
          })}
        >
          <StatusIcon status={task.status} />
        </Pressable>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`${task.title}, ${t(statusKey(task.status))}${project ? `, ${project.name}` : ''}`}
          onPress={openTask}
          style={({ pressed }) => ({
            flex: 1,
            minHeight: 64,
            paddingVertical: 16,
            paddingRight: 12,
            gap: 4,
            opacity: pressed ? 0.6 : 1,
          })}
        >
          <Text
            style={[
              type.body,
              {
                color: task.status === 'done' ? colors.secondary : colors.text,
                fontWeight: '500',
              },
            ]}
          >
            {task.title}
          </Text>
          {(project ||
            task.dueAt ||
            task.priority === 'P0' ||
            task.priority === 'P1') && (
            <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
              {(task.priority === 'P0' || task.priority === 'P1') && (
                <Text
                  style={[
                    type.label,
                    {
                      color:
                        task.priority === 'P0'
                          ? colors.error
                          : colors.secondary,
                    },
                  ]}
                >
                  {task.priority}
                </Text>
              )}
              {project && (
                <Text style={[type.label, { color: colors.secondary }]}>
                  {project.name}
                </Text>
              )}
              {task.dueAt && (
                <Text
                  style={[
                    type.label,
                    {
                      color:
                        new Date(task.dueAt).getTime() < now &&
                        task.status !== 'done'
                          ? colors.stale
                          : colors.secondary,
                    },
                  ]}
                >
                  {dateLabel(task.dueAt, locale)}
                </Text>
              )}
            </View>
          )}
          {agent && (
            <>
              <Text
                numberOfLines={2}
                style={[type.body, { color: colors.secondary }]}
              >
                {report?.summary ?? t('noReport')}
              </Text>
              {report && (
                <Text
                  style={[
                    type.label,
                    { color: stale ? colors.stale : colors.secondary },
                  ]}
                >
                  {report.agent} · {reportAge(report.createdAt, t, now)}
                  {stale ? ` · ${t('stale')}` : ''}
                </Text>
              )}
            </>
          )}
        </Pressable>
      </View>
      <ErrorMessage message={action.error} />
    </View>
  );
}
