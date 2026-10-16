import { useState } from 'react';
import { FlatList, SectionList, Text, View } from 'react-native';
import { router } from 'expo-router';
import { dayKey } from '../i18n';
import {
  Button,
  Choice,
  Empty,
  Field,
  IconButton,
  Label,
  Page,
} from '../ui/components';
import { TaskRow } from '../ui/TaskRow';
import { SyncLine } from '../ui/SyncLine';
import { type, useUI } from '../ui/theme';

export function TaskList({ mode }: { mode: 'mine' | 'agents' }) {
  const { snapshot, colors, t } = useUI();
  const [filter, setFilter] = useState<'active' | 'completed'>('active');
  const agent = mode === 'agents';
  const archived = new Set(
    snapshot.projects.filter((p) => p.archivedAt).map((p) => p.id),
  );
  const tasks = snapshot.tasks.filter(
    (task) =>
      task.source === (agent ? 'agent' : 'user') &&
      !task.parentId &&
      !archived.has(task.projectId ?? '') &&
      (filter === 'completed'
        ? ['done', 'cancelled'].includes(task.status)
        : !['done', 'cancelled'].includes(task.status)),
  );
  const today = dayKey(new Date());
  const sections =
    agent || filter === 'completed'
      ? [
          {
            title: t(filter === 'completed' ? 'completed' : 'active'),
            data: tasks.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt)),
          },
        ]
      : [
          {
            title: t('overdue'),
            data: tasks.filter(
              (task) => task.dueAt && dayKey(task.dueAt) < today,
            ),
          },
          {
            title: t('today'),
            data: tasks.filter(
              (task) => task.dueAt && dayKey(task.dueAt) === today,
            ),
          },
          {
            title: t('later'),
            data: tasks.filter(
              (task) => !task.dueAt || dayKey(task.dueAt) > today,
            ),
          },
        ].filter((section) => section.data.length);
  return (
    <Page
      title={t(mode)}
      scroll={false}
      right={
        <IconButton
          name="search"
          label={t('search')}
          onPress={() => router.push('/search')}
        />
      }
    >
      <SectionList
        sections={sections}
        keyExtractor={(task) => task.id}
        keyboardShouldPersistTaps="handled"
        contentContainerStyle={{
          flexGrow: 1,
          paddingBottom: 100,
          width: '100%',
          maxWidth: 900,
          alignSelf: 'center',
        }}
        ListHeaderComponent={
          <View
            style={{
              paddingHorizontal: 20,
              paddingTop: 8,
              paddingBottom: 12,
              gap: 16,
            }}
          >
            <SyncLine />
            <Choice
              label={t('status')}
              selected={filter}
              onSelect={setFilter}
              options={[
                { value: 'active', label: t('active') },
                { value: 'completed', label: t('completed') },
              ]}
            />
            {agent && <Label secondary>{t('agentHelp')}</Label>}
          </View>
        }
        renderSectionHeader={({ section }) => (
          <Text
            accessibilityRole="header"
            style={[
              type.label,
              {
                color: colors.secondary,
                paddingHorizontal: 24,
                paddingTop: 24,
                paddingBottom: 8,
                backgroundColor: colors.surface,
              },
            ]}
          >
            {section.title} · {section.data.length}
          </Text>
        )}
        renderItem={({ item }) => <TaskRow task={item} agent={agent} />}
        stickySectionHeadersEnabled={false}
        ListEmptyComponent={
          <Empty
            title={t(
              filter === 'completed'
                ? 'noCompleted'
                : agent
                  ? 'noAgents'
                  : 'noTasks',
            )}
            body={t(
              filter === 'completed'
                ? 'noCompletedBody'
                : agent
                  ? 'noAgentsBody'
                  : 'noTasksBody',
            )}
            icon={agent ? 'robot' : 'inbox'}
          />
        }
      />
      {!agent && (
        <View style={{ position: 'absolute', bottom: 20, right: 20 }}>
          <Button
            label={t('newTask')}
            testID="task-add"
            icon="plus"
            onPress={() => router.push('/task/new')}
          />
        </View>
      )}
    </Page>
  );
}
export function SearchScreen() {
  const { snapshot, t } = useUI();
  const [query, setQuery] = useState('');
  const term = query.trim().toLocaleLowerCase();
  const tasks = term
    ? snapshot.tasks.filter((task) =>
        [
          task.title,
          task.notes,
          snapshot.projects.find((p) => p.id === task.projectId)?.name ?? '',
        ].some((text) => text.toLocaleLowerCase().includes(term)),
      )
    : [];
  return (
    <Page title={t('search')} back scroll={false}>
      <View style={{ paddingHorizontal: 20, paddingVertical: 12 }}>
        <Field
          autoFocus
          value={query}
          onChangeText={setQuery}
          placeholder={t('searchTasks')}
          returnKeyType="search"
          clearButtonMode="while-editing"
        />
      </View>
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
        ListEmptyComponent={
          <Empty
            title={t(term ? 'noResults' : 'searchTasks')}
            body={term ? t('noResultsBody') : undefined}
            icon="search"
          />
        }
      />
    </Page>
  );
}
