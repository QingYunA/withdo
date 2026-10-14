import { afterEach, describe, expect, test, vi } from "vitest";
import { notifyReminder, reminderText } from "./notify";

const reminder = { task_id: 7, title: "Call mum", project_name: "Home" };

afterEach(() => vi.unstubAllGlobals());

describe("reminderText", () => {
  test("includes the project when there is one", () => {
    expect(reminderText(reminder)).toEqual({ title: "Reminder: Call mum", body: "Home" });
    expect(reminderText({ ...reminder, project_name: null })).toEqual({ title: "Reminder: Call mum", body: "Inbox" });
  });
});

describe("notifyReminder", () => {
  test("uses the Tauri notification plugin after checking permission", async () => {
    const plugin = {
      isPermissionGranted: vi.fn(async () => false),
      requestPermission: vi.fn(async () => "granted" as const),
      sendNotification: vi.fn(),
    };
    await notifyReminder(reminder, { tauri: true, loadPlugin: async () => plugin });
    expect(plugin.requestPermission).toHaveBeenCalled();
    expect(plugin.sendNotification).toHaveBeenCalledWith({ title: "Reminder: Call mum", body: "Home" });
  });

  test("does not send when the user denies permission", async () => {
    const plugin = {
      isPermissionGranted: vi.fn(async () => false),
      requestPermission: vi.fn(async () => "denied" as const),
      sendNotification: vi.fn(),
    };
    await notifyReminder(reminder, { tauri: true, loadPlugin: async () => plugin });
    expect(plugin.sendNotification).not.toHaveBeenCalled();
  });

  test("uses the Notification API in the browser when permission is already granted", async () => {
    const ctor = vi.fn();
    Object.assign(ctor, { permission: "granted" });
    vi.stubGlobal("Notification", ctor);
    await notifyReminder(reminder, { tauri: false });
    expect(ctor).toHaveBeenCalledWith("Reminder: Call mum", { body: "Home" });
  });

  test("is a silent no-op in a browser without notification support or permission", async () => {
    vi.stubGlobal("Notification", undefined);
    await expect(notifyReminder(reminder, { tauri: false })).resolves.toBeUndefined();
    const ctor = vi.fn();
    Object.assign(ctor, { permission: "default" });
    vi.stubGlobal("Notification", ctor);
    await notifyReminder(reminder, { tauri: false });
    expect(ctor).not.toHaveBeenCalled();
  });
});
