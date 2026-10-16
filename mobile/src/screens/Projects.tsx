import { useState } from 'react';
import { Alert, FlatList, Pressable, Text, View } from 'react-native';
import { router } from 'expo-router';
import { appStore } from '../data';
import {
  Button,
  Choice,
  Empty,
  ErrorMessage,
  Field,
  IconButton,
  Label,
  Page,
  Row,
  useAction,
} from '../ui/components';
import { Icon } from '../ui/icons';
import { TaskRow } from '../ui/TaskRow';
import { type, useUI } from '../ui/theme';

export function ProjectsScreen() {
  const { snapshot, colors, t } = useUI();
  const [archived, setArchived] = useState<'active' | 'archived'>('active');
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const action = useAction();
  const create = () =>
    void action.run(async () => {
      if (!name.trim()) throw new Error(t('projectRequired'));
      const project = await appStore.createProject(name.trim());
      setName('');
      setCreating(false);
      router.push({ pathname: '/project/[id]', params: { id: project.id } });
    });
  const projects = snapshot.projects
    .filter((project) =>
      archived === 'archived' ? !!project.archivedAt : !project.archivedAt,
    )
    .sort((a, b) => a.name.localeCompare(b.name));
  return (
    <Page
      title={t('projects')}
      scroll={false}
      right={
        <IconButton
          name={creating ? 'close' : 'plus'}
          label={t(creating ? 'cancel' : 'newProject')}
          onPress={() => setCreating(!creating)}
        />
      }
    >
      <FlatList
        data={projects}
        keyExtractor={(project) => project.id}
        keyboardShouldPersistTaps="handled"
        contentContainerStyle={{
          flexGrow: 1,
          paddingBottom: 32,
          width: '100%',
          maxWidth: 900,
          alignSelf: 'center',
        }}
        ListHeaderComponent={
          <View style={{ padding: 20, gap: 16 }}>
            <Choice
              label={t('projects')}
              selected={archived}
              onSelect={setArchived}
              options={[
                { value: 'active', label: t('active') },
                { value: 'archived', label: t('archived') },
              ]}
            />
            {creating && (
              <>
                <Field
                  label={t('projectName')}
                  autoFocus
                  value={name}
                  onChangeText={setName}
                  onSubmitEditing={create}
                  returnKeyType="done"
                />
                <Button label={t('add')} busy={action.busy} onPress={create} />
              </>
            )}
            <ErrorMessage message={action.error} />
          </View>
        }
        renderItem={({ item }) => (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={item.name}
            onPress={() =>
              router.push({
                pathname: '/project/[id]',
                params: { id: item.id },
              })
            }
            style={({ pressed }) => ({
              flexDirection: 'row',
              alignItems: 'center',
              gap: 16,
              minHeight: 72,
              paddingHorizontal: 24,
              paddingVertical: 16,
              backgroundColor: pressed ? colors.subtle : 'transparent',
            })}
          >
            <Icon name="folder" color={colors.primary} />
            <Text
              style={[
                type.body,
                { color: colors.text, fontWeight: '500', flex: 1 },
              ]}
            >
              {item.name}
            </Text>
            <Text style={[type.label, { color: colors.secondary }]}>
              {
                snapshot.tasks.filter(
                  (task) =>
                    task.projectId === item.id &&
                    !task.parentId &&
                    !['done', 'cancelled'].includes(task.status),
                ).length
              }
            </Text>
            <Icon name="chevron-right" />
          </Pressable>
        )}
        ListEmptyComponent={
          <Empty
            title={t(archived === 'archived' ? 'noArchived' : 'noProjects')}
            body={archived === 'active' ? t('noProjectsBody') : undefined}
            icon="folder"
            action={
              archived === 'active' && !creating ? (
                <Button
                  label={t('newProject')}
                  kind="tonal"
                  onPress={() => setCreating(true)}
                />
              ) : undefined
            }
          />
        }
      />
    </Page>
  );
}
export function ProjectScreen({ id }: { id: string }) {
  const { snapshot, t } = useUI();
  const project = snapshot.projects.find((item) => item.id === id);
  const [renaming, setRenaming] = useState(false);
  const [name, setName] = useState(project?.name ?? '');
  const [filter, setFilter] = useState<'active' | 'completed'>('active');
  const action = useAction();
  if (!project)
    return (
      <Page title={t('project')} back>
        <Empty title={t('projectNotFound')} body={t('projectNotFoundBody')} />
      </Page>
    );
  const tasks = snapshot.tasks.filter(
    (task) =>
      task.projectId === id &&
      !task.parentId &&
      (filter === 'active'
        ? !['done', 'cancelled'].includes(task.status)
        : ['done', 'cancelled'].includes(task.status)),
  );
  const archive = () =>
    Alert.alert(t('archiveQuestion'), t('archiveBody'), [
      { text: t('cancel'), style: 'cancel' },
      {
        text: t('archive'),
        onPress: () =>
          void action.run(async () => {
            await appStore.updateProject(id, {
              archivedAt: new Date().toISOString(),
            });
            router.back();
          }),
      },
    ]);
  return (
    <Page
      title={project.name}
      back
      scroll={false}
      right={
        <IconButton
          name="plus"
          label={t('newTask')}
          onPress={() =>
            router.push({ pathname: '/task/new', params: { projectId: id } })
          }
        />
      }
    >
      <FlatList
        data={tasks}
        keyExtractor={(task) => task.id}
        keyboardShouldPersistTaps="handled"
        contentContainerStyle={{
          flexGrow: 1,
          paddingBottom: 32,
          width: '100%',
          maxWidth: 900,
          alignSelf: 'center',
        }}
        renderItem={({ item }) => (
          <TaskRow task={item} agent={item.source === 'agent'} />
        )}
        ListHeaderComponent={
          <View style={{ paddingHorizontal: 20, paddingBottom: 16, gap: 16 }}>
            {project.archivedAt && <Label secondary>{t('archived')}</Label>}
            <Row style={{ flexWrap: 'wrap' }}>
              <Button
                label={t('rename')}
                kind="text"
                onPress={() => {
                  setName(project.name);
                  setRenaming(!renaming);
                }}
              />
              <Button
                label={t(project.archivedAt ? 'restore' : 'archive')}
                kind="text"
                disabled={action.busy}
                onPress={() =>
                  project.archivedAt
                    ? void action.run(() =>
                        appStore.updateProject(id, { archivedAt: null }),
                      )
                    : archive()
                }
              />
            </Row>
            {renaming && (
              <>
                <Field
                  label={t('projectName')}
                  value={name}
                  onChangeText={setName}
                  autoFocus
                />
                <Button
                  label={t('save')}
                  busy={action.busy}
                  onPress={() =>
                    void action.run(async () => {
                      if (!name.trim()) throw new Error(t('projectRequired'));
                      await appStore.updateProject(id, { name: name.trim() });
                      setRenaming(false);
                    })
                  }
                />
              </>
            )}
            <ErrorMessage message={action.error} />
            <Choice
              label={t('status')}
              selected={filter}
              onSelect={setFilter}
              options={[
                { value: 'active', label: t('active') },
                { value: 'completed', label: t('completed') },
              ]}
            />
          </View>
        }
        ListEmptyComponent={
          <Empty
            title={t(filter === 'completed' ? 'noCompleted' : 'noTasks')}
            body={t(filter === 'completed' ? 'noCompletedBody' : 'noTasksBody')}
            action={
              <Button
                label={t('newTask')}
                kind="tonal"
                onPress={() =>
                  router.push({
                    pathname: '/task/new',
                    params: { projectId: id },
                  })
                }
              />
            }
          />
        }
      />
    </Page>
  );
}
