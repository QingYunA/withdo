# Changelog

Each release has a section here. When updates are on (`docs/plans/m10-auto-update.md`), `mygo build`
reads the update window's notes from the section of the version (`## 0.5.0`) and fails without one.

## 0.12.0

No database change on your computer; the app updates itself from 0.11.0.

- **Sign up and sign in are now two separate ways in.** Signing up asks for your email and a
  password, then a 6-digit code from your email to verify it. Signing in is your email and
  password.
- **Forgot your password?** Reset it right in the app with a code from your email; no browser page.
- **Signed up with a code before?** Your account has no password you know yet: choose Sign in, then
  Forgot password, and set one. Google sign-in works as before.
- Code emails now come in the app's language (English or Simplified Chinese).
- Android gets the same Sign in, Sign up and Forgot password steps.

## 0.11.1

No database change; the app updates itself from 0.11.0.

- **A new Settings window.** The panes are in a sidebar: Account, General, Categories, Agents, Model
  and About. Your agents show as tiles with their pixel avatars, and you connect one with one click.
  Pick your model from a grid of providers.
- The project menu in the sidebar and the milestone menus now match the app's look, light and dark,
  instead of the system's.
- Dragging the window by its top no longer shakes.
- Copy and paste shortcuts work even when no text field has the focus.
- The Account page no longer shows the official sync server's address, and on Windows it calls
  this computer a PC, not a Mac.

## 0.11.0

No database change. Install this version from the DMG or the installer: 0.10.2 and older look for
updates in a repository that no longer gets them. From this version on, updates come from
github.com/QingYunA/withdo.

- **ADHD Todo is now Withdo.** Same app, same data, new name and a new pixel icon.
- Your coding agents are moved to the new MCP server name `withdo` by themselves. If you allowed
  `mcp__adhd-todo__*` tools in an agent's permissions, allow `mcp__withdo__*` instead. The command
  line is `withdo` now; `adhd-core` still works.
- On a Mac, delete the old `ADHD Todo` app after installing Withdo.
- Home shows your idle agents in a tray with a sleeping robot, and Summary is back in the sidebar.
- Withdo is open source under the AGPL-3.0: github.com/QingYunA/withdo.

## 0.10.2

No database change; the app updates itself from 0.10.0.

- Done today keeps the task's name in view when an agent left a long note, and the row no longer
  runs under the agents beside it.
- Sharper picture and send buttons in the input.
- The app now looks for updates every four hours instead of once a day.

## 0.10.1

No database change; the desktop app updates itself from 0.10.0.

- **Android gets the new Today page.** The date is the title, your agents' cards come first, then
  Today, Next, a folded No date list and Done today. The input stays at the bottom where your thumb
  reaches it.
- **Do today** in the long-press menu on Android.
- New colors on Android, light and dark, matching the desktop.

## 0.10.0

No database change. From 0.9.0 the app updates itself; on Windows, quit your coding agents before
you restart into the new version.

- **A calmer Today page.** The date is the title and the input sits across the top: type or say
  what you need, several things in one sentence, some for you and some for an agent.
- **Sorted by time only.** Today, then Next (what is due in the coming week), then a folded No date
  list for undated tasks with no project. Undated tasks of a project stay on the project page.
- **Do today.** Right-click a task (or press T) to make it due today.
- **Agents beside your list** as cards, each with one main button. Drag a task onto them to hand it
  over; the drop area shows only while you drag.
- New colors throughout, light and dark, with thinner lines and neutral selection.

## 0.9.0

No database change: upgrading from 0.8.1 is installing the app. On Windows, quit ADHD Todo and your
coding agents before you install.

- **The app updates itself.** This is the last version to install by hand. From now on ADHD Todo
  checks for new versions once a day (Settings > General > Software updates; `Check for Updates…` in
  the app menu), downloads them, checks their signature and asks you to restart. On Android a banner
  says when a new version is out; Update downloads it and opens the installer.
- Downloads moved to a public page: github.com/QingYunA/adhd-todo-releases.

## 0.8.1

No database change: upgrading from 0.8.0 is installing the app. On Windows, quit ADHD Todo and your
coding agents before you install.

- **Signed in to the wrong account?** If this Mac (or phone) was syncing with one account and you
  sign in with another, the app now asks first: sign in with the old account again, or move this
  device to the new one, keeping every task. A device that belongs to another account says so, with
  a Sign out button, instead of "server not responding".
- **Link Google to your account** in Settings > Account (the Sync screen on Android), so Continue
  with Google signs in to the account you already have.
- "Server not responding" now shows only when the server really cannot be reached.

## 0.8.0

No database change: upgrading from 0.7.3 is installing the app. On Windows, quit ADHD Todo and your
coding agents before you install.

- **Home opens with one big input.** The date, `What do you want to do today?` and a large input at
  the top; your lists under it. What your agents are doing sits in a column on the right, with how
  long ago each one reported; drag a to-do onto it to hand it to an agent. Only project pages keep
  the small input at the bottom; `⌘K` anywhere else brings you to the big one.
- **Three places in the sidebar: Today, Agents and Calendar** (`⌘1`–`⌘3`), then your projects.
  - Tasks without a project show in Today or Later by their date; those due after next week fold at
    the end of Later.
  - Completed tasks: `Show all` at the end of any Done section opens their history.
  - Today's summary: `See today ›` beside Done today.
  - Categories are colored chips on your tasks; click one to see only that category. Manage them in
    Settings > General.
  - The Agents page is grouped by agent, with tasks waiting to start under Queued.

## 0.7.3

No database change: upgrading from 0.7.2 is installing the app. On Windows, quit ADHD Todo and your
coding agents before you install.

- **The old sign-in service is gone.** A device that was still signed in through it shows the sign-in
  card once: sign in again with the same email (a code, a password or Google) and everything is
  there. Versions before 0.7.0 can no longer sign in; update them.
- On Android, Continue with Google opens inside the app instead of switching to the browser.

## 0.7.2

No database change: upgrading from 0.7.1 is installing the app. On Windows, quit ADHD Todo and your
coding agents before you install.

- **Continue with Google.** Settings > Account (the Sync screen on Android) has a Google button above
  the email field. It opens your browser once; choose your Google account there and the app goes on
  by itself. A Google account with the same email as an existing account signs in to that account.

## 0.7.1

No database change: upgrading from 0.7.0 is installing the app. On Windows, quit ADHD Todo and your
coding agents before you install.

- **Sign up from the app.** Settings > Account (the Sync screen on Android) asks only for your email
  and mails you a 6-digit code; a new email gets an account right there. A password is optional:
  `Use a password instead` keeps it, and `Forgot password` mails a reset link.
- **Voice fills the input instead of sending it,** so you can fix a word before you press Return.
- **The input looks the same before and after you click it:** only its border changes, and nothing
  in it moves.

## 0.7.0

No database change: upgrading from 0.6.4 is installing the app. On Windows, quit ADHD Todo and your
coding agents before you install.

- **Say it in one sentence.** Without a model connected, the input still reads what you type: days
  and times (`明天九点`, `next friday`), `提醒我` / `remind me` for a reminder, `#category`, and `!` for
  important. `，还有` / `, and also` splits one sentence into several tasks. One Undo takes them all
  back.
- **Voice input** on macOS: click the microphone at the right of the input, speak, click again. The
  words become the input and are sent as if typed. It needs you signed in to sync.
- **The menu bar icon.** Click it, or press `⌥ Space` anywhere (`Alt+Space` on Windows, where the tray
  icon opens it too), to add a task and see today's tasks and your agents without opening the window.
- **Your tasks are simpler.** They have three states, To do, Doing and Done, and a star for the
  important ones instead of P0 to P3. A task's pane shows when, where, notes and pictures, and a
  button to hand it to an agent. An agent's tasks keep all their states and details.
- **Settings has four panes:** Account, General, Agents and Advanced. Your own model key and the
  less used switches moved to Advanced.
- **Accounts moved to a new sign-in service.** Your email and password are the same and nothing needs
  doing: devices signed in before keep syncing, and the next sign-in uses the new service.
- Drawing is faster: the window redraws only what changed.

## 0.6.4

No database change: upgrading from 0.6.3 is installing the app. On Windows, quit ADHD Todo and your
coding agents before you install.

- **Fixed: no new task or report could be saved** on a synced computer, with `UNIQUE constraint
  failed: tasks.id`. An `adhd-core` started before 0.5.2 and still running took ids the sync had
  reserved, and every later save tried the same id. Saving now skips an id that is already taken.
  Restart your coding agents after updating, so none of them keeps running an old `adhd-core`.

## 0.6.3

No database change: upgrading from 0.6.2 is installing the app. On Windows, quit ADHD Todo and your
coding agents before you install.

- **Android: the task menu.** Long-press a task, or tap ⋯ on a task's page, to change its status,
  priority, category or project, set a reminder, open its pull request, or delete it (Undo brings
  it back for six seconds). A task's page opens the same pages from its rows.
- **Android: fixed** a task's page staying blank on a phone with no projects.
- The desktop's reminder presets now come from the same code as the phone's; they are unchanged.

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
