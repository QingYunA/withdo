# Changelog

Each release has a section here. When updates are on (`docs/plans/m10-auto-update.md`), `mygo build`
reads the update window's notes from the section of the version (`## 0.5.0`) and fails without one.

## 0.6.2

No database change: upgrading from 0.6.1 is installing the app. On Windows, quit ADHD Todo and your
coding agents before you install.

- **Signing in is faster**: about 2 seconds instead of 5. The app sends one request and the sync
  server does the rest.
- While signing in, the button says what is happening: checking your account, then downloading
  your tasks.
- The Android app's version is 0.6.2.

## 0.6.1

No database change: upgrading from 0.6.0 is installing the app. On Windows, quit ADHD Todo and your
coding agents before you install.

- **Sign in to sync inside the app.** Settings › Sync asks for your account and password; no
  browser opens. Accounts are made by the server's admin. An account with two-step verification
  cannot sign in this way yet.
- The Android app signs in the same way, and its version now matches the desktop's (0.6.1).

## 0.6.0

No database change: upgrading from 0.5.2 or 0.5.3 is installing the app (from an older version the
database moves to schema 14 on the first start). On Windows, quit ADHD Todo and your coding agents
before you install. Restart your coding agents afterwards.

- **Sync between devices.** Settings has a Sync tab: sign in through the browser and this computer
  syncs its tasks, projects and agents' reports through todo.sri-ai.xyz with your other computers
  and the Android app. When both this computer and the account have tasks, you choose to merge them
  into the account (task numbers change to the account's, agents use the new ones the next time
  they report) or to use the account's (this computer's data is backed up first). The first
  upload shows its progress; the tab lists your devices and says when a sync failed and when it
  retries. The sidebar's Settings row shows a red cloud only when sync needs you.
- A reply you write on your phone shows in the task's activity as yours.
- **Windows:** a tray icon (left click opens the window, right click has Open, Settings and Exit),
  one instance (a second launch brings the window back), a File menu, and keyboard hints written
  as Ctrl and Shift.
- **Windows:** Terminal, Editor and the agent's app open (Windows Terminal or cmd; Cursor, VS Code
  or Zed); folders open in File Explorer, pictures in their default app; the GitHub CLI hint names
  winget; Copilot, Cline and Zed are found where Windows keeps their settings.

## 0.5.3

No database change: upgrading from 0.5.2 is installing the app. On Windows, quit ADHD Todo and your
coding agents before you install, or the installer cannot replace the files they hold open. Restart
your coding agents afterwards.

- **adhd-core comes with the Windows app.** The installer puts the console `adhd-core.exe` next to
  `ADHD Todo.exe`, and Settings > Agents connects agents to it, one click or the copied command.
  The separate `adhd-core.exe` zip is no longer published. An agent you connected to the zip's copy
  keeps running that copy: remove the adhd-todo server from it and connect it again from Settings >
  Agents.
- On macOS, running the app straight from the disk image, or from a copy macOS runs from a random
  folder (App Translocation), no longer points `~/.local/bin/adhd-core` at a path that goes away;
  the link is made once the app runs from Applications.
- Toasts and the bulk bar move with the composer as it grows or shrinks, in the same frame.
- The command line gains `adhd-core sync`, groundwork for syncing between devices; the app has no
  sync settings yet.

## 0.5.2

The database moves from schema 13 to 14 (the groundwork of syncing between devices; nothing syncs
yet), which 0.5.1's adhd-core cannot read. On macOS the app is adhd-core, so installing it is enough;
on Windows, replace `adhd-core.exe` with the one of this release too. Restart your coding agents
afterwards.

- **Windows again.** Each release now ships the Windows x64 installer and `adhd-core.exe` next to
  the macOS files.
- **Undo everywhere.** Every change that offers Undo in a toast also answers Cmd+Z (Ctrl+Z on
  Windows), including deleting a task, detaching a picture and dragging a single card, which had no
  undo before.
- Undoing one of the assistant's memory edits takes back only that edit; notes it kept later, and
  your own edits to `assistant-memory.md`, stay.
- Amp and Kiro are drawn as their own marks instead of the generic robot.

## 0.5.1

No database change: upgrading from 0.5.0 is installing the app. Restart your coding agents afterwards
to get the new MCP tool.

- **Agents see a task's pictures.** `get_task` lists them with their files, and the new
  `get_task_image` shows one, scaled down when it is large.
- **Your own tasks stay yours.** Tasks you wrote that are in no project are no longer visible to
  coding agents over MCP; hand one to an agent and it can see it.
- **The assistant remembers.** Corrections and preferences you give it are kept in
  `assistant-memory.md` in the data folder (Settings > Assistant opens it), each with Undo.
- **The assistant files its tasks.** When you name no category it picks one and says so in a toast,
  with Undo.
- The assistant's box sits at the bottom of the page, like a chat.
- Rename a project by double-clicking its title. The sub-rows of Mine and Agents fold, and a new
  project's repository field is folded until you need it.
- In a project shown as two columns, each column has its own Select.
- DeepSeek Harness is the whale everywhere.
- **A smaller download.** adhd-core is now the app's own program instead of a second copy inside it,
  so the DMG is about 8 MB smaller. `~/.local/bin/adhd-core` is repointed on the next launch.

## 0.5.0

Upgrading from 0.4.0: quit your coding agents first and back up `~/Library/Application Support/adhd-todo/`.
The database moves from schema 11 to 13, which 0.4.0's adhd-core cannot read; on its first launch
the app points `~/.local/bin/adhd-core` at its own copy (the old file is kept as `adhd-core.old`).

- **adhd-core comes with the app.** The app links `~/.local/bin/adhd-core`, the path your coding
  agents run, to the copy inside it, so installing the app brings both. A coding agent keeps running
  the old adhd-core until it is restarted; restart it after installing. `adhd-core --version` prints
  the version.
- **Your own categories**, with 12 x 12 pixel icons in color, an icon editor, and import from a
  picture or SVG. Work and Life stay as the built-in two.
- **Sub-projects**: a project can hang under another in the sidebar (right-click > New sub-project).
- **Recent changes**, a page off by default (Settings > General > Pages) that reads your projects'
  repositories with `gh` and has the assistant explain what merged.
- **Summary**: quiet sound effects while the day is shipped (Settings > General can mute them), and
  six people in the scene, with figures of their own for Gemini, Cursor, Copilot and OpenCode.
- Fixes: completing a task from a grouped list by its hover glyph, the Summary ledger's two-digit
  counts, undoing a delete keeps subtasks under their parent, and the test suite and SQLite on Windows.

## 0.4.0

- **Summary page**: the day's and the week's shipping, and an end-of-day reminder.
- **Pictures on tasks**, and **selecting several tasks** to act on them together, each with undo.
- **Pages from agents**: an agent can attach an HTML report or a design draft to its report.
- **System notifications** when an agent finishes or gets stuck, or a pull request fails or merges.
- **Agents set themselves up**: `adhd-core guide` prints what any coding agent needs to add the MCP
  server, and Settings > Agents has more harnesses.
- The pixel look all through the interface.
