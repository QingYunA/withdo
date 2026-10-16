import * as Notifications from 'expo-notifications';
import { AndroidImportance, IosAuthorizationStatus } from 'expo-notifications';
import { Platform } from 'react-native';
import type { Task } from '../model';

const prefix = 'adhd-todo-task-';
const channelId = 'task-reminders';
let serial: Promise<unknown> = Promise.resolve();
let handlerInstalled = false;

/** Reconcile against the OS list, which also survives process restarts. */
export function reconcileReminders(tasks: Task[], enabled: boolean, requestPermission: boolean): Promise<string | null> {
  const work = serial.then(async () => {
    if (!handlerInstalled) {
      Notifications.setNotificationHandler({
        handleNotification: async () => ({ shouldShowBanner: true, shouldShowList: true, shouldPlaySound: true, shouldSetBadge: false }),
      });
      handlerInstalled = true;
    }
    if (Platform.OS === 'android') {
      await Notifications.setNotificationChannelAsync(channelId, { name: 'Task reminders', importance: AndroidImportance.DEFAULT });
    }
    let permission = await Notifications.getPermissionsAsync();
    if (enabled && requestPermission && !permission.granted && permission.canAskAgain) permission = await Notifications.requestPermissionsAsync();
    const permitted = permission.granted || permission.ios?.status === IosAuthorizationStatus.PROVISIONAL;
    const desired = new Map((enabled && permitted ? tasks : [])
      .filter(task => task.remindAt && Date.parse(task.remindAt) > Date.now() && task.status !== 'done' && task.status !== 'cancelled')
      .map(task => [`${prefix}${task.id}`, task]));
    for (const pending of await Notifications.getAllScheduledNotificationsAsync()) {
      if (!pending.identifier.startsWith(prefix)) continue;
      const task = desired.get(pending.identifier);
      if (task && pending.content.title === task.title && pending.content.body === (task.notes || 'Task reminder') && pending.content.data?.remindAt === task.remindAt) {
        desired.delete(pending.identifier);
        continue;
      }
      await Notifications.cancelScheduledNotificationAsync(pending.identifier);
    }
    for (const [identifier, task] of desired) {
      await Notifications.scheduleNotificationAsync({
        identifier,
        content: { title: task.title, body: task.notes || 'Task reminder', sound: 'default', data: { taskId: task.id, remindAt: task.remindAt } },
        trigger: { type: Notifications.SchedulableTriggerInputTypes.DATE, date: new Date(task.remindAt!), channelId },
      });
    }
    return enabled && !permitted ? 'Notifications are disabled. Allow notifications in system settings to receive task reminders.' : null;
  });
  serial = work.catch(() => undefined);
  return work;
}
