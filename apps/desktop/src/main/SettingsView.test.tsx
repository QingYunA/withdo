import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, test, vi } from "vitest";

const shell = vi.hoisted(() => ({
  getShortcut: vi.fn<() => Promise<string | null>>(async () => "Alt+Space"),
  setShortcut: vi.fn<(s: string) => Promise<string>>(async (s) => s),
}));
vi.mock("../platform/shell", () => shell);

import { SettingsView } from "./SettingsView";

beforeEach(() => {
  vi.clearAllMocks();
  shell.getShortcut.mockResolvedValue("Alt+Space");
  shell.setShortcut.mockImplementation(async (s) => s);
});

describe("SettingsView", () => {
  test("shows the current shortcut in Mac terms", async () => {
    render(<SettingsView />);
    expect(await screen.findByText("Option+Space")).toBeInTheDocument();
  });

  test("records a new shortcut and saves it", async () => {
    render(<SettingsView />);
    await userEvent.click(await screen.findByRole("button", { name: "Change shortcut" }));
    expect(screen.getByText(/Press the new shortcut/)).toBeInTheDocument();
    await userEvent.keyboard("{Meta>}{Shift>}k{/Shift}{/Meta}");
    await waitFor(() => expect(shell.setShortcut).toHaveBeenCalledWith("Command+Shift+K"));
    expect(await screen.findByText("Cmd+Shift+K")).toBeInTheDocument();
  });

  test("ignores a key press with no modifier and cancels on Escape", async () => {
    render(<SettingsView />);
    await userEvent.click(await screen.findByRole("button", { name: "Change shortcut" }));
    await userEvent.keyboard("k");
    expect(shell.setShortcut).not.toHaveBeenCalled();
    await userEvent.keyboard("{Escape}");
    expect(screen.queryByText(/Press the new shortcut/)).not.toBeInTheDocument();
  });

  test("shows why a shortcut was refused and keeps the old one", async () => {
    shell.setShortcut.mockRejectedValue("shortcut Alt+K is already taken");
    render(<SettingsView />);
    await userEvent.click(await screen.findByRole("button", { name: "Change shortcut" }));
    await userEvent.keyboard("{Alt>}k{/Alt}");
    expect(await screen.findByText(/already taken/)).toBeInTheDocument();
    expect(screen.getByText("Option+Space")).toBeInTheDocument();
  });

  test("explains that the shortcut only exists in the desktop app when run in a browser", async () => {
    shell.getShortcut.mockResolvedValue(null);
    render(<SettingsView />);
    expect(await screen.findByText(/desktop app/)).toBeInTheDocument();
  });
});
