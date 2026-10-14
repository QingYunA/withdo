import { isTauri } from "./endpoint";
import type { Reminder } from "./types";

interface NotificationPlugin {
  isPermissionGranted(): Promise<boolean>;
  requestPermission(): Promise<"granted" | "denied" | "default">;
  sendNotification(options: { title: string; body?: string }): void;
}

export interface NotifyDeps {
  readonly tauri: boolean;
  readonly loadPlugin?: () => Promise<NotificationPlugin>;
}

export function reminderText(reminder: Reminder): { title: string; body: string } {
  return { title: `Reminder: ${reminder.title}`, body: reminder.project_name ?? "Inbox" };
}

async function viaTauri(text: ReturnType<typeof reminderText>, load: () => Promise<NotificationPlugin>): Promise<void> {
  const plugin = await load();
  const granted = (await plugin.isPermissionGranted()) || (await plugin.requestPermission()) === "granted";
  if (granted) plugin.sendNotification(text);
}

function viaBrowser(text: ReturnType<typeof reminderText>): void {
  if (typeof Notification === "undefined" || Notification.permission !== "granted") return;
  new Notification(text.title, { body: text.body });
}

/** System notification inside Tauri; the Notification API in a browser (only if already allowed). */
export async function notifyReminder(
  reminder: Reminder,
  deps: NotifyDeps = { tauri: isTauri() },
): Promise<void> {
  const text = reminderText(reminder);
  if (deps.tauri) return viaTauri(text, deps.loadPlugin ?? (() => import("@tauri-apps/plugin-notification")));
  viaBrowser(text);
}
