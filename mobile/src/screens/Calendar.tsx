import { useMemo, useState } from 'react';
import {
  FlatList,
  Pressable,
  ScrollView,
  Text,
  useWindowDimensions,
  View,
} from 'react-native';
import { router } from 'expo-router';
import { dateLabel, dayKey } from '../i18n';
import { Button, Empty, IconButton, Page, Row } from '../ui/components';
import { TaskRow } from '../ui/TaskRow';
import { type, useUI } from '../ui/theme';

export function CalendarScreen() {
  const { snapshot, colors, t, locale } = useUI();
  const { width } = useWindowDimensions();
  const expanded = width >= 1000;
  const [month, setMonth] = useState(
    () => new Date(new Date().getFullYear(), new Date().getMonth(), 1),
  );
  const [selected, setSelected] = useState(() => dayKey(new Date()));
  const cells = useMemo(() => {
    const start = new Date(month.getFullYear(), month.getMonth(), 1);
    start.setDate(start.getDate() - ((start.getDay() + 6) % 7));
    return Array.from({ length: 42 }, (_, index) => {
      const day = new Date(start);
      day.setDate(start.getDate() + index);
      return day;
    });
  }, [month]);
  const datedTasks = snapshot.tasks.filter(
    (task) => task.dueAt && !['done', 'cancelled'].includes(task.status),
  );
  const counts = new Map<string, number>();
  datedTasks.forEach((task) => {
    const key = dayKey(task.dueAt!);
    counts.set(key, (counts.get(key) ?? 0) + 1);
  });
  const tasks = snapshot.tasks
    .filter((task) => task.dueAt && dayKey(task.dueAt) === selected)
    .sort(
      (a, b) => (a.status === 'done' ? 1 : 0) - (b.status === 'done' ? 1 : 0),
    );
  const monthTitle = new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'long',
  }).format(month);
  const days = Array.from({ length: 7 }, (_, index) =>
    new Intl.DateTimeFormat(locale, { weekday: 'narrow' }).format(
      new Date(2026, 0, 5 + index),
    ),
  );
  const calendar = (
    <View
      style={{
        width: expanded ? 440 : '100%',
        paddingHorizontal: 16,
        paddingVertical: 8,
      }}
    >
      <Row style={{ marginBottom: 12 }}>
        <IconButton
          name="chevron-left"
          label={t('previousMonth')}
          onPress={() =>
            setMonth(new Date(month.getFullYear(), month.getMonth() - 1, 1))
          }
        />
        <Text
          accessibilityRole="header"
          style={[
            type.title,
            { color: colors.text, flex: 1, textAlign: 'center' },
          ]}
        >
          {monthTitle}
        </Text>
        <IconButton
          name="chevron-right"
          label={t('nextMonth')}
          onPress={() =>
            setMonth(new Date(month.getFullYear(), month.getMonth() + 1, 1))
          }
        />
      </Row>
      <ScrollView
        horizontal
        contentContainerStyle={{ flexGrow: 1 }}
        showsHorizontalScrollIndicator={false}
      >
        <View style={{ minWidth: 336, flex: 1 }}>
          <View style={{ flexDirection: 'row' }}>
            {days.map((day, index) => (
              <Text
                key={index}
                style={[
                  type.label,
                  {
                    color: colors.secondary,
                    flex: 1,
                    textAlign: 'center',
                    paddingBottom: 12,
                  },
                ]}
              >
                {day}
              </Text>
            ))}
          </View>
          <View style={{ flexDirection: 'row', flexWrap: 'wrap' }}>
            {cells.map((day) => {
              const key = dayKey(day),
                count = counts.get(key) ?? 0,
                isSelected = key === selected;
              const isToday = key === dayKey(new Date());
              return (
                <Pressable
                  key={key}
                  accessibilityRole="button"
                  accessibilityLabel={`${new Intl.DateTimeFormat(locale, { dateStyle: 'full' }).format(day)}, ${count} ${t('task')}`}
                  accessibilityState={{ selected: isSelected }}
                  onPress={() => {
                    setSelected(key);
                    if (day.getMonth() !== month.getMonth())
                      setMonth(new Date(day.getFullYear(), day.getMonth(), 1));
                  }}
                  style={({ pressed }) => ({
                    width: `${100 / 7}%`,
                    minHeight: 56,
                    paddingVertical: 8,
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: 4,
                    backgroundColor: isSelected
                      ? colors.selected
                      : pressed
                        ? colors.subtle
                        : 'transparent',
                    borderRadius: 16,
                  })}
                >
                  <Text
                    style={[
                      type.body,
                      {
                        color:
                          isSelected || isToday
                            ? colors.primary
                            : day.getMonth() === month.getMonth()
                              ? colors.text
                              : colors.secondary,
                        fontWeight: isToday || isSelected ? '700' : '400',
                      },
                    ]}
                  >
                    {day.getDate()}
                  </Text>
                  <View
                    style={{
                      width: 4,
                      height: 4,
                      borderRadius: 2,
                      backgroundColor: count ? colors.primary : 'transparent',
                    }}
                  />
                </Pressable>
              );
            })}
          </View>
        </View>
      </ScrollView>
      <Button
        label={t('goToday')}
        kind="text"
        onPress={() => {
          const now = new Date();
          setSelected(dayKey(now));
          setMonth(new Date(now.getFullYear(), now.getMonth(), 1));
        }}
      />
    </View>
  );
  const heading = (
    <Row style={{ paddingHorizontal: 24, paddingTop: 20 }}>
      <Text
        accessibilityRole="header"
        style={[type.title, { color: colors.text, flex: 1 }]}
      >
        {dateLabel(`${selected}T12:00:00`, locale)}
      </Text>
      <IconButton
        name="plus"
        label={t('newTask')}
        onPress={() =>
          router.push({
            pathname: '/task/new',
            params: { dueAt: new Date(`${selected}T23:59:00`).toISOString() },
          })
        }
      />
    </Row>
  );
  return (
    <Page title={t('calendar')} scroll={false}>
      <View
        style={{
          flex: 1,
          flexDirection: expanded ? 'row' : 'column',
          maxWidth: 1100,
          width: '100%',
          alignSelf: 'center',
        }}
      >
        {expanded && calendar}
        <FlatList
          style={{ flex: 1 }}
          data={tasks}
          keyExtractor={(task) => task.id}
          contentContainerStyle={{ flexGrow: 1, paddingBottom: 32 }}
          renderItem={({ item }) => (
            <TaskRow task={item} agent={item.source === 'agent'} />
          )}
          ListHeaderComponent={
            <>
              {!expanded && calendar}
              {heading}
            </>
          }
          ListEmptyComponent={
            <Empty
              title={t('noDayTasks')}
              body={t('noDayTasksBody')}
              icon="calendar"
            />
          }
        />
      </View>
    </Page>
  );
}
