import { useState } from 'react';
import { Alert, Linking, Pressable, Text, View } from 'react-native';
import { DateTimePickerAndroid } from '@react-native-community/datetimepicker';
import { router, usePreventRemove } from 'expo-router';
import { appStore } from '../data';
import { dateLabel, statusKey } from '../i18n';
import type { TaskPatch, TaskStatus } from '../model';
import {
  Button,
  Choice,
  Empty,
  ErrorMessage,
  Field,
  Label,
  Page,
  Row,
  Section,
  useAction,
} from '../ui/components';
import { Icon, StatusIcon } from '../ui/icons';
import { latestReport, reportAge, TaskRow } from '../ui/TaskRow';
import { type, useUI } from '../ui/theme';

const statuses: TaskStatus[] = [
  'backlog',
  'todo',
  'doing',
  'blocked',
  'review',
  'done',
  'cancelled',
];
function DateField({
  label,
  value,
  onChange,
  withTime = false,
}: {
  label: string;
  value: string | null;
  onChange: (value: string | null) => void;
  withTime?: boolean;
}) {
  const { t, colors, locale } = useUI();
  const pick = () => {
    const valueDate = value
      ? new Date(value)
      : new Date(Date.now() + 3_600_000);
    DateTimePickerAndroid.open({
      value: valueDate,
      mode: 'date',
      onValueChange: (_event, selected) => {
        if (!withTime) {
          selected.setHours(23, 59, 0, 0);
          onChange(selected.toISOString());
          return;
        }
        DateTimePickerAndroid.open({
          value: selected,
          mode: 'time',
          onValueChange: (_timeEvent, time) => {
            onChange(time.toISOString());
          },
        });
      },
    });
  };
  return (
    <View style={{ gap: 8 }}>
      <Text style={[type.label, { color: colors.secondary }]}>{label}</Text>
      <Row>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`${label}: ${value ? dateLabel(value, locale, withTime) : t('none')}`}
          onPress={pick}
          style={({ pressed }) => ({
            minHeight: 56,
            flex: 1,
            flexDirection: 'row',
            alignItems: 'center',
            gap: 12,
            backgroundColor: colors.subtle,
            padding: 16,
            borderRadius: 12,
            opacity: pressed ? 0.6 : 1,
          })}
        >
          <Icon name={withTime ? 'bell' : 'calendar'} />
          <Label>
            {value ? dateLabel(value, locale, withTime) : t('none')}
          </Label>
        </Pressable>
        {value && (
          <Button
            label={t('clear')}
            kind="text"
            onPress={() => onChange(null)}
          />
        )}
      </Row>
    </View>
  );
}
export function TaskEditor({
  taskId,
  projectId,
  parentId,
  dueAt,
}: {
  taskId?: string;
  projectId?: string;
  parentId?: string;
  dueAt?: string;
}) {
  const { snapshot, t, colors, locale, now } = useUI();
  const task = snapshot.tasks.find((task) => task.id === taskId);
  const [initial] = useState<TaskPatch>(() => ({
    title: task?.title ?? '',
    notes: task?.notes ?? '',
    status: task?.status ?? 'todo',
    priority: task?.priority ?? 'P2',
    projectId: task?.projectId ?? projectId ?? null,
    parentId: task?.parentId ?? parentId ?? null,
    dueAt:
      task?.dueAt ??
      (dueAt && Number.isFinite(Date.parse(dueAt)) ? dueAt : null),
    remindAt: task?.remindAt ?? null,
  }));
  const [draft, setDraft] = useState(initial);
  const [allowLeave, setAllowLeave] = useState(false);
  const [subtask, setSubtask] = useState('');
  const action = useAction();
  const subtaskAction = useAction();
  const dirty = JSON.stringify(initial) !== JSON.stringify(draft);
  const disablePrevention = usePreventRemove(
    !allowLeave && dirty,
    ({ repeat }) =>
      Alert.alert(t('unsaved'), t('unsavedBody'), [
        { text: t('cancel'), style: 'cancel' },
        {
          text: t('discard'),
          style: 'destructive',
          onPress: () => {
            setAllowLeave(true);
            repeat();
          },
        },
      ]),
  );
  const goBack = () => {
    disablePrevention();
    setAllowLeave(true);
    if (router.canGoBack()) router.back();
    else router.replace('/');
  };
  const patch = (value: TaskPatch) => setDraft((old) => ({ ...old, ...value }));
  const save = () =>
    void action.run(async () => {
      if (!draft.title?.trim()) throw new Error(t('titleRequired'));
      if (
        draft.remindAt &&
        draft.remindAt !== task?.remindAt &&
        new Date(draft.remindAt).getTime() <= Date.now()
      )
        throw new Error(t('reminderPast'));
      const input = { ...draft, title: draft.title.trim() };
      if (taskId) {
        const changed = Object.fromEntries(
          Object.entries(input).filter(
            ([key, value]) => value !== initial[key as keyof TaskPatch],
          ),
        ) as TaskPatch;
        await appStore.updateTask(taskId, changed);
      } else await appStore.createTask(input);
      goBack();
    });
  const remove = () =>
    Alert.alert(t('deleteQuestion'), t('deleteBody'), [
      { text: t('cancel'), style: 'cancel' },
      {
        text: t('delete'),
        style: 'destructive',
        onPress: () =>
          void action.run(async () => {
            await appStore.deleteTask(taskId!);
            goBack();
          }),
      },
    ]);
  if (taskId && !task)
    return (
      <Page title={t('task')} back>
        <Empty title={t('taskNotFound')} body={t('taskNotFoundBody')} />
      </Page>
    );
  const report = task ? latestReport(task.id, snapshot.reports) : undefined;
  const parent = snapshot.tasks.find((item) => item.id === draft.parentId);
  const projects = snapshot.projects.filter(
    (project) => !project.archivedAt || project.id === draft.projectId,
  );
  const bottom = (
    <>
      <ErrorMessage message={action.error} />
      <Button
        label={t('save')}
        testID="task-save"
        busy={action.busy}
        disabled={!!task && !dirty}
        onPress={save}
      />
    </>
  );
  return (
    <Page title={t(task ? 'editTask' : 'newTask')} back bottom={bottom}>
      {parent && (
        <Button
          label={`${t('parentTask')}: ${parent.title}`}
          kind="text"
          onPress={() =>
            router.push({ pathname: '/task/[id]', params: { id: parent.id } })
          }
        />
      )}
      <Field
        label={t('title')}
        testID="task-title-input"
        placeholder={t('titlePlaceholder')}
        value={draft.title}
        onChangeText={(title) => patch({ title })}
        autoFocus={!task}
        multiline
        style={{ minHeight: 80 }}
      />
      <Field
        label={t('notes')}
        testID="task-notes-input"
        placeholder={t('notesPlaceholder')}
        value={draft.notes}
        onChangeText={(notes) => patch({ notes })}
        multiline
      />
      <Section title={t('status')}>
        <View style={{ gap: 8 }}>
          {statuses.map((status) => (
            <Pressable
              key={status}
              accessibilityRole="radio"
              accessibilityState={{ selected: draft.status === status }}
              accessibilityLabel={t(statusKey(status))}
              onPress={() => patch({ status })}
              style={({ pressed }) => ({
                minHeight: 48,
                paddingHorizontal: 12,
                paddingVertical: 12,
                flexDirection: 'row',
                alignItems: 'center',
                gap: 12,
                borderRadius: 12,
                backgroundColor:
                  draft.status === status
                    ? colors.selected
                    : pressed
                      ? colors.subtle
                      : 'transparent',
              })}
            >
              <StatusIcon status={status} />
              <Label style={{ flex: 1 }}>{t(statusKey(status))}</Label>
              {draft.status === status && (
                <Icon name="check" color={colors.primary} />
              )}
            </Pressable>
          ))}
        </View>
      </Section>
      <Section title={t('priority')}>
        <Choice
          label={t('priority')}
          selected={draft.priority ?? 'P2'}
          options={(['P0', 'P1', 'P2', 'P3'] as const).map((value) => ({
            value,
            label: value,
          }))}
          onSelect={(priority) => patch({ priority })}
        />
      </Section>
      <Section title={t('project')}>
        <Choice
          label={t('project')}
          selected={draft.projectId ?? ''}
          options={[
            { value: '', label: t('noProject') },
            ...projects.map((project) => ({
              value: project.id,
              label: project.name,
            })),
          ]}
          onSelect={(projectId) => patch({ projectId: projectId || null })}
        />
      </Section>
      <Section title={t('schedule')}>
        <DateField
          label={t('due')}
          value={draft.dueAt ?? null}
          onChange={(dueAt) => patch({ dueAt })}
        />
        <DateField
          label={t('reminder')}
          value={draft.remindAt ?? null}
          onChange={(remindAt) => patch({ remindAt })}
          withTime
        />
        {draft.remindAt && !snapshot.settings.remindersEnabled && (
          <Label secondary>{t('enableReminders')}</Label>
        )}
      </Section>
      {task && (
        <Section title={t('subtasks')}>
          {snapshot.tasks
            .filter((child) => child.parentId === task.id)
            .map((child) => (
              <TaskRow key={child.id} task={child} />
            ))}
          <Field
            label={t('addSubtask')}
            value={subtask}
            onChangeText={setSubtask}
            onSubmitEditing={() =>
              void subtaskAction.run(async () => {
                if (!subtask.trim()) return;
                await appStore.createTask({
                  title: subtask.trim(),
                  parentId: task.id,
                  projectId: task.projectId,
                });
                setSubtask('');
              })
            }
            returnKeyType="done"
          />
          <Button
            label={t('add')}
            icon="plus"
            kind="tonal"
            disabled={!subtask.trim()}
            busy={subtaskAction.busy}
            onPress={() =>
              void subtaskAction.run(async () => {
                await appStore.createTask({
                  title: subtask.trim(),
                  parentId: task.id,
                  projectId: task.projectId,
                });
                setSubtask('');
              })
            }
          />
          <ErrorMessage message={subtaskAction.error} />
        </Section>
      )}
      {task?.source === 'agent' && (
        <Section title={t('latestReport')}>
          <Label>{report?.summary ?? t('noReport')}</Label>
          {report && (
            <>
              <Label secondary>
                {report.agent} · {reportAge(report.createdAt, t, now)} ·{' '}
                {dateLabel(report.createdAt, locale, true)}
              </Label>
              {report.nextStep ? (
                <>
                  <Label secondary>{t('nextStep')}</Label>
                  <Label>{report.nextStep}</Label>
                </>
              ) : null}
            </>
          )}
          {task.branch ? (
            <Label secondary>
              {t('branch')}: {task.branch}
            </Label>
          ) : null}
          {task.prUrl && (
            <Button
              label={t('openPR')}
              kind="text"
              icon="external-link"
              onPress={() =>
                void action.run(async () => {
                  const url = new URL(task.prUrl);
                  if (url.protocol !== 'https:' && url.protocol !== 'http:')
                    throw new Error(t('invalidUrl'));
                  await Linking.openURL(task.prUrl);
                })
              }
            />
          )}
        </Section>
      )}
      {task && (
        <View style={{ marginVertical: 24 }}>
          <Button
            label={t('delete')}
            kind="danger"
            icon="trash"
            disabled={action.busy}
            onPress={remove}
          />
        </View>
      )}
    </Page>
  );
}
