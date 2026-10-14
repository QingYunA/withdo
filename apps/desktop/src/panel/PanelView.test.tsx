import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Overview } from "../core/types";
import { fakeClient, renderWithCore } from "../test/render";
import { makeOverviewTask, makeReport, makeTask } from "../test/fixtures";

const shell = vi.hoisted(() => ({
  hidePanel: vi.fn(async () => {}),
  openTaskInMain: vi.fn(async () => {}),
  onShellEvent: vi.fn(() => () => {}),
}));
vi.mock("../platform/shell", () => shell);

import { PanelView } from "./PanelView";

const hoursAgo = (h: number) => new Date(Date.now() - h * 3_600_000).toISOString();
const inHours = (h: number) => new Date(Date.now() + h * 3_600_000).toISOString();

const overview: Overview = {
  generated_at: new Date().toISOString(),
  stale_hours: 48,
  projects: [
    {
      project: { id: 1, name: "widgets", repo_slug: "acme/widgets" },
      tasks: [
        makeOverviewTask({
          id: 10,
          title: "Fix search ranking",
          status: "blocked",
          branch: "fix/search",
          latest_report: makeReport({ summary: "Needs a tokenizer decision", created_at: hoursAgo(5) }),
        }),
        makeOverviewTask({
          id: 11,
          title: "Rewrite install guide",
          status: "doing",
          branch: "docs/install",
          stale: true,
          updated_at: hoursAgo(72),
          latest_report: makeReport({ summary: "Drafted macOS section", created_at: hoursAgo(72) }),
        }),
      ],
    },
    { project: null, tasks: [makeOverviewTask({ id: 12, title: "Loose agent task", status: "review" })] },
  ],
};

const mine = [
  makeTask({ id: 20, title: "Book dentist", due_at: inHours(30) }),
  makeTask({ id: 21, title: "Reply to design feedback", remind_at: inHours(1.5), priority: "P1" }),
  makeTask({ id: 22, title: "An agent todo", source: "agent", status: "todo" }),
];

function setup(overrides = {}) {
  const createTask = vi.fn(async () => makeTask({ id: 99, title: "new" }));
  const client = fakeClient({ overview: async () => overview, tasks: async () => mine, createTask, ...overrides });
  const view = renderWithCore(<PanelView />, { client });
  return { ...view, createTask };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("PanelView sections", () => {
  test("groups agent work by project and shows status, branch, summary, age", async () => {
    setup();
    const widgets = await screen.findByRole("group", { name: "widgets" });
    expect(within(widgets).getByText("Fix search ranking")).toBeInTheDocument();
    expect(within(widgets).getByText("fix/search")).toBeInTheDocument();
    expect(within(widgets).getByText("Needs a tokenizer decision")).toBeInTheDocument();
    expect(within(widgets).getByText("5h")).toBeInTheDocument();
    expect(within(widgets).getByRole("img", { name: "Blocked" })).toBeInTheDocument();
    const inbox = screen.getByRole("group", { name: "Inbox" });
    expect(within(inbox).getByText("Loose agent task")).toBeInTheDocument();
  });

  test("flags stale agent tasks with a badge and leaves fresh ones alone", async () => {
    setup();
    const stale = await screen.findByText("Stale 3d");
    expect(stale.closest("button")).toHaveTextContent("Rewrite install guide");
    expect(screen.getAllByText(/^Stale/)).toHaveLength(1);
  });

  test("Mine lists only the user's own tasks, soonest first", async () => {
    setup();
    const section = await screen.findByRole("region", { name: "Mine" });
    const rows = await within(section).findAllByRole("button");
    expect(rows.map((r) => r.textContent)).toEqual([
      expect.stringContaining("Reply to design feedback"),
      expect.stringContaining("Book dentist"),
    ]);
    expect(within(section).getByText("Remind in 1h")).toBeInTheDocument();
    expect(within(section).queryByText("An agent todo")).not.toBeInTheDocument();
  });

  test("explains how to connect Claude Code and how to add tasks when everything is empty", async () => {
    setup({ overview: async () => ({ ...overview, projects: [] }), tasks: async () => [] });
    expect(await screen.findByText(/No agents are reporting yet/)).toBeInTheDocument();
    expect(screen.getByText(/claude mcp add/)).toBeInTheDocument();
    expect(await screen.findByText(/Nothing on your list/)).toBeInTheDocument();
  });

  test("shows an error line when the core cannot be read", async () => {
    setup({
      overview: async () => {
        throw new Error("Cannot reach adhd-core");
      },
    });
    expect(await screen.findByText(/Cannot reach adhd-core/)).toBeInTheDocument();
  });
});

describe("PanelView input", () => {
  test("is focused when the panel opens", async () => {
    setup();
    expect(screen.getByRole("textbox", { name: /new task/i })).toHaveFocus();
  });

  test("Enter creates an inbox task with the typed title, clears the box and confirms", async () => {
    const { createTask } = setup();
    const input = screen.getByRole("textbox", { name: /new task/i });
    await userEvent.type(input, "  Call the plumber {Enter}");
    await waitFor(() => expect(createTask).toHaveBeenCalledWith({ title: "Call the plumber" }));
    expect(input).toHaveValue("");
    expect(await screen.findByText("Added to Inbox")).toBeInTheDocument();
  });

  test("Enter on a blank box does nothing", async () => {
    const { createTask } = setup();
    await userEvent.type(screen.getByRole("textbox", { name: /new task/i }), "   {Enter}");
    expect(createTask).not.toHaveBeenCalled();
  });

  test("keeps the text and shows the error when saving fails", async () => {
    setup({
      createTask: async () => {
        throw new Error("title: required");
      },
    });
    const input = screen.getByRole("textbox", { name: /new task/i });
    await userEvent.type(input, "oops{Enter}");
    expect(await screen.findByText("title: required")).toBeInTheDocument();
    expect(input).toHaveValue("oops");
  });

  test("Escape hides the panel", async () => {
    setup();
    await userEvent.keyboard("{Escape}");
    expect(shell.hidePanel).toHaveBeenCalled();
  });
});

describe("PanelView keyboard and clicks", () => {
  test("clicking a task opens it in the main window and hides the panel", async () => {
    setup();
    await userEvent.click(await screen.findByText("Fix search ranking"));
    expect(shell.openTaskInMain).toHaveBeenCalledWith(10);
    expect(shell.hidePanel).toHaveBeenCalled();
  });

  test("arrow keys move through rows and Enter on an empty box opens the selected task", async () => {
    setup();
    await screen.findByText("Fix search ranking");
    await userEvent.keyboard("{ArrowDown}");
    expect(screen.getByText("Fix search ranking").closest("button")).toHaveAttribute("aria-current", "true");
    await userEvent.keyboard("{ArrowDown}{Enter}");
    expect(shell.openTaskInMain).toHaveBeenCalledWith(11);
  });
});
