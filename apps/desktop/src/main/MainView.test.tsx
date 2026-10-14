import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Milestone, ProjectWithCounts, Report, Task } from "../core/types";
import { fakeClient, renderWithCore } from "../test/render";
import { makeReport, makeTask } from "../test/fixtures";
import { useHashRoute } from "../ui/use-hash-route";

const shell = vi.hoisted(() => ({
  openExternal: vi.fn(async () => {}),
  getShortcut: vi.fn(async () => null),
  setShortcut: vi.fn(async () => ""),
  onShellEvent: vi.fn(() => () => {}),
}));
vi.mock("../platform/shell", () => shell);

import { MainView } from "./MainView";

const counts = { backlog: 0, todo: 2, doing: 1, blocked: 0, review: 1, done: 1, cancelled: 0 };
const project = (id: number, name: string): ProjectWithCounts => ({
  id, name, repo_slug: `acme/${name}`, repo_path: null, created_at: "", updated_at: "", archived_at: null, counts,
});
const projects = [project(1, "widgets"), project(2, "docs")];
const milestones: Milestone[] = [{ id: 5, project_id: 1, title: "v1.0", due_at: null, status: "open", created_at: "" }];

const widgetTasks: Task[] = [
  makeTask({ id: 1, project_id: 1, title: "Add login form", status: "doing", source: "agent", branch: "feat/login", milestone_id: 5, priority: "P1" }),
  makeTask({ id: 2, project_id: 1, title: "Cache invalidation", status: "review", source: "agent", pr_url: "https://github.com/acme/widgets/pull/42", pr_number: 42 }),
  makeTask({ id: 3, project_id: 1, title: "Reply to feedback", status: "todo" }),
  makeTask({ id: 4, project_id: 1, title: "Ship changelog", status: "done" }),
  makeTask({ id: 9, project_id: 1, title: "Dropped idea", status: "cancelled" }),
];
const inboxTasks: Task[] = [makeTask({ id: 7, project_id: null, title: "Buy milk", status: "todo" })];

const reports: Report[] = [
  makeReport({ id: 1, task_id: 1, summary: "Started on the form", created_at: "2026-10-13T09:00:00.000Z" }),
  makeReport({ id: 2, task_id: 1, summary: "Form wired to the API", next_step: "Write tests", status: "doing", created_at: "2026-10-14T09:00:00.000Z" }),
];

function Harness() {
  const route = useHashRoute();
  if (route.name !== "main") return null;
  return <MainView route={route} />;
}

function setup(overrides = {}) {
  const updateTask = vi.fn(async (id: number, patch: object) => ({ ...makeTask({ id }), ...patch }) as Task);
  const client = fakeClient({
    projects: async () => projects,
    tasks: vi.fn(async (filter = {}) => (filter.inbox ? inboxTasks : filter.project_id === 1 ? widgetTasks : [])),
    milestones: async (id: number) => (id === 1 ? milestones : []),
    task: async (id: number) => ({ task: [...widgetTasks, ...inboxTasks].find((t) => t.id === id)!, reports }),
    updateTask,
    ...overrides,
  });
  return { ...renderWithCore(<Harness />, { client }), updateTask };
}

const column = (name: string) => screen.getByRole("region", { name });

beforeEach(() => {
  vi.clearAllMocks();
  window.location.hash = "#/main";
});

describe("sidebar and board", () => {
  test("lists Inbox and projects, opens the first project and groups cards by status column", async () => {
    setup();
    const nav = await screen.findByRole("navigation");
    await within(nav).findByRole("button", { name: /docs/ });
    expect(within(nav).getAllByRole("button").map((b) => b.textContent)).toEqual(
      expect.arrayContaining([expect.stringContaining("Inbox"), expect.stringContaining("widgets"), expect.stringContaining("docs")]),
    );
    expect(await screen.findByRole("heading", { name: "widgets" })).toBeInTheDocument();
    expect(within(column("Doing")).getByText("Add login form")).toBeInTheDocument();
    expect(within(column("Review")).getByText("Cache invalidation")).toBeInTheDocument();
    expect(within(column("Todo")).getByText("Reply to feedback")).toBeInTheDocument();
    expect(within(column("Done")).getByText("Ship changelog")).toBeInTheDocument();
    expect(screen.queryByText("Dropped idea")).not.toBeInTheDocument();
    expect(within(column("Doing")).getByText("1")).toBeInTheDocument(); // column count
  });

  test("switching to the inbox loads inbox tasks", async () => {
    setup();
    await screen.findByText("Add login form");
    await userEvent.click(within(screen.getByRole("navigation")).getByRole("button", { name: /Inbox/ }));
    expect(await screen.findByText("Buy milk")).toBeInTheDocument();
    expect(screen.queryByText("Add login form")).not.toBeInTheDocument();
  });

  test("cards show priority and a PR link that opens externally without opening the drawer", async () => {
    setup();
    const login = (await screen.findByText("Add login form")).closest("[data-card]") as HTMLElement;
    expect(within(login).getByText("P1")).toBeInTheDocument();
    expect(within(login).getByText("feat/login")).toBeInTheDocument();
    await userEvent.click(within(column("Review")).getByRole("link", { name: "PR #42" }));
    expect(shell.openExternal).toHaveBeenCalledWith("https://github.com/acme/widgets/pull/42");
    expect(screen.queryByRole("complementary")).not.toBeInTheDocument();
  });

  test("filters by milestone", async () => {
    setup();
    await screen.findByText("Add login form");
    await userEvent.selectOptions(screen.getByRole("combobox", { name: "Milestone" }), "v1.0");
    expect(screen.getByText("Add login form")).toBeInTheDocument();
    expect(screen.queryByText("Reply to feedback")).not.toBeInTheDocument();
    await userEvent.selectOptions(screen.getByRole("combobox", { name: "Milestone" }), "All milestones");
    expect(screen.getByText("Reply to feedback")).toBeInTheDocument();
  });

  test("an empty project explains how tasks arrive", async () => {
    setup({ tasks: async () => [] });
    expect(await screen.findByText(/No tasks in this project yet/)).toBeInTheDocument();
  });
});

describe("task drawer", () => {
  test("clicking a card opens the drawer with details and reports, newest first", async () => {
    setup();
    await userEvent.click(await screen.findByText("Add login form"));
    const drawer = await screen.findByRole("complementary");
    expect(window.location.hash).toBe("#/main/task/1");
    expect(await within(drawer).findByDisplayValue("Add login form")).toBeInTheDocument();
    const items = within(drawer).getAllByRole("listitem");
    expect(items[0]).toHaveTextContent("Form wired to the API");
    expect(items[0]).toHaveTextContent("Next: Write tests");
    expect(items[1]).toHaveTextContent("Started on the form");
  });

  test("a deep link opens the drawer and selects the task's project", async () => {
    window.location.hash = "#/main/task/7";
    setup();
    expect(await screen.findByDisplayValue("Buy milk")).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: "Inbox" })).toBeInTheDocument();
  });

  test("saving sends only the changed fields", async () => {
    window.location.hash = "#/main/task/3";
    const { updateTask } = setup();
    const title = await screen.findByLabelText("Title");
    const save = screen.getByRole("button", { name: "Save" });
    expect(save).toBeDisabled();
    await userEvent.clear(title);
    await userEvent.type(title, "Reply to design feedback");
    await userEvent.selectOptions(screen.getByLabelText("Priority"), "P0");
    await userEvent.click(save);
    await waitFor(() => expect(updateTask).toHaveBeenCalledWith(3, { title: "Reply to design feedback", priority: "P0" }));
  });

  test("an empty title is rejected before anything is sent", async () => {
    window.location.hash = "#/main/task/3";
    const { updateTask } = setup();
    await userEvent.clear(await screen.findByLabelText("Title"));
    await userEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(screen.getByText("Title is required")).toBeInTheDocument();
    expect(updateTask).not.toHaveBeenCalled();
  });

  test("opens the PR in the browser and closes with the close button", async () => {
    window.location.hash = "#/main/task/2";
    setup();
    await userEvent.click(await screen.findByRole("button", { name: /Open PR #42/ }));
    expect(shell.openExternal).toHaveBeenCalledWith("https://github.com/acme/widgets/pull/42");
    await userEvent.click(screen.getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.queryByRole("complementary")).not.toBeInTheDocument());
    expect(window.location.hash).toBe("#/main");
  });

  test("keeps unsaved edits when the task changes underneath", async () => {
    window.location.hash = "#/main/task/3";
    const { queryClient } = setup();
    const title = await screen.findByLabelText("Title");
    await userEvent.type(title, " (draft)");
    await act(async () => {
      await queryClient.invalidateQueries();
    });
    expect(screen.getByLabelText("Title")).toHaveValue("Reply to feedback (draft)");
  });
});

describe("settings", () => {
  test("the settings route shows the shortcut section", async () => {
    window.location.hash = "#/main/settings";
    setup();
    expect(await screen.findByRole("heading", { name: "Settings" })).toBeInTheDocument();
  });
});
