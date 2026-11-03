package xyz.sriai.adhdtodo.ui

import androidx.activity.compose.BackHandler
import android.content.Context
import android.content.res.Resources
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.util.Log
import androidx.browser.customtabs.CustomTabColorSchemeParams
import androidx.browser.customtabs.CustomTabsIntent
import androidx.compose.ui.graphics.toArgb
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.snapshotFlow
import androidx.compose.runtime.saveable.listSaver
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.coroutines.withTimeoutOrNull
import xyz.sriai.adhdtodo.BuildConfig
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Account
import xyz.sriai.adhdtodo.data.AgentInfo
import xyz.sriai.adhdtodo.data.AgentsPage
import xyz.sriai.adhdtodo.data.CalendarPage
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.CoreApi
import xyz.sriai.adhdtodo.data.Device
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.ModelPhase
import xyz.sriai.adhdtodo.data.Milestone
import xyz.sriai.adhdtodo.data.PageJson
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.ProjectEntry
import xyz.sriai.adhdtodo.data.ProjectPage
import xyz.sriai.adhdtodo.data.RemindPreset
import xyz.sriai.adhdtodo.data.SignInState
import xyz.sriai.adhdtodo.data.SyncStatus
import xyz.sriai.adhdtodo.data.Task
import xyz.sriai.adhdtodo.data.TaskPatch
import xyz.sriai.adhdtodo.data.TaskPage
import xyz.sriai.adhdtodo.data.UpdateUi
import xyz.sriai.adhdtodo.platform.Reminders
import xyz.sriai.adhdtodo.core.Core_
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import java.time.Instant
import java.time.LocalDate
import java.time.YearMonth
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.temporal.ChronoUnit

/**
 * The app shell: the visible page, the composer pinned above the bottom bar, and the stack of
 * subpages (a project, a task, Sync). The Activity owns the core and its listener; every page is
 * (re)read on a change, off the UI thread, and a failed read leaves the last good page on screen.
 */
@Composable
fun App(
    core: Core_,
    revision: Int,
    syncStatusJson: String,
    initialRoute: String? = null,
    demoSyncState: String? = null,
    onInitialRouteConsumed: () -> Unit = {},
    update: UpdateUi = UpdateUi(),
    onCheckUpdate: () -> Unit = {},
    onUpdate: () -> Unit = {},
    onDismissUpdate: () -> Unit = {},
) {
    val context = LocalContext.current
    val resources = context.resources
    val tokens = LocalTokens.current
    val words = remember(resources.configuration) { Words(resources) }
    val api = remember(core) { CoreApi(core) }
    val scope = rememberCoroutineScope()

    val routeSaver = listSaver<List<String>, String>(save = { it }, restore = { it })
    var stack by rememberSaveable(stateSaver = routeSaver) { mutableStateOf(listOf("home")) }
    var tabName by rememberSaveable { mutableStateOf(Tab.MINE.name) }
    var tick by remember { mutableIntStateOf(0) }
    // The Calendar tab: the month shown, the selected day (both kept while on other tabs) and the page read for the month.
    var calMonthKey by rememberSaveable { mutableIntStateOf(YearMonth.now().key()) }
    var calDayEpoch by rememberSaveable { mutableLongStateOf(LocalDate.now().toEpochDay()) }
    var calendarPage by remember { mutableStateOf<CalendarPage?>(null) }

    var mine by remember { mutableStateOf<MinePage?>(null) }
    var mineAgents by remember { mutableStateOf<Map<Long, AgentInfo>>(emptyMap()) }
    var agents by remember { mutableStateOf<AgentsPage?>(null) }
    var agentsInfo by remember { mutableStateOf<Map<Long, AgentInfo>>(emptyMap()) }
    var projectEntries by remember { mutableStateOf<List<ProjectEntry>?>(null) }
    var projectPage by remember { mutableStateOf<ProjectPage?>(null) }
    var archivedProjects by remember { mutableStateOf<List<ProjectEntry>>(emptyList()) }
    var settingsData by remember { mutableStateOf(SettingsData()) }
    var newProject by remember { mutableStateOf(false) }
    var newCategoryFor by remember { mutableStateOf<Long?>(null) }
    var taskPage by remember { mutableStateOf<TaskPage?>(null) }
    var projects by remember { mutableStateOf<List<Project>>(emptyList()) }
    var categories by remember { mutableStateOf<List<Category>>(emptyList()) }
    var account by remember { mutableStateOf<Account?>(null) }
    var devices by remember { mutableStateOf<List<Device>>(emptyList()) }
    var liveStatus by remember { mutableStateOf<SyncStatus?>(null) }

    var draft by rememberSaveable { mutableStateOf("") }
    var sending by remember { mutableStateOf(false) }
    var notice by remember { mutableStateOf<String?>(null) }
    var noticeUndo by remember { mutableStateOf<(() -> Unit)?>(null) }
    var noticeId by remember { mutableIntStateOf(0) }
    var noDateOpen by rememberSaveable { mutableStateOf(false) }
    var highlightId by remember { mutableStateOf(0L) }
    var toast by remember { mutableStateOf<UndoToast?>(null) }
    var replyTo by remember { mutableStateOf<Item?>(null) }
    var connect by remember { mutableStateOf(false) }
    var menuData by remember { mutableStateOf<TaskMenuData?>(null) }
    var menuPage by remember { mutableStateOf(TaskMenuPage.Main) }
    var catFilter by remember { mutableStateOf(0L) }
    var searchQuery by rememberSaveable { mutableStateOf("") }
    var taskMilestones by remember { mutableStateOf<List<Milestone>>(emptyList()) }
    var composeFocus by remember { mutableIntStateOf(0) }
    // The assistant behind the box (v44): the official model's state, the turn in flight and its card.
    val assistant = remember { AssistantState() }
    // Select mode (v43): who is ticked (null: off), the rows the page draws, and the bar's two pickers.
    var selection by remember { mutableStateOf<Set<Long>?>(null) }
    var selectOrder by remember { mutableStateOf<List<Long>>(emptyList()) }
    var bulkSheet by remember { mutableStateOf<String?>(null) }
    var bulkProjects by remember { mutableStateOf<List<Project>>(emptyList()) }
    val summary = remember { SummaryHolder() }

    var signingIn by remember { mutableStateOf(false) }
    var choosing by remember { mutableStateOf(false) }
    var serverEditing by remember { mutableStateOf(false) }
    var serverDraft by remember { mutableStateOf("") }
    var serverError by remember { mutableStateOf(false) }
    var serverSaving by remember { mutableStateOf(false) }
    var syncMessage by remember { mutableStateOf<String?>(null) }
    var choose by remember { mutableStateOf(SignInState()) }
    var flow by remember { mutableStateOf(SignInFlow()) }
    var googleAvailable by remember { mutableStateOf(false) }
    var otherAccount by remember { mutableStateOf(SignInState()) }
    var otherChoosing by remember { mutableStateOf(false) }
    var googleRow by remember { mutableStateOf(GoogleLinkRow()) }

    val route = stack.last()
    val tab = runCatching { Tab.valueOf(tabName) }.getOrDefault(Tab.MINE)
    val locale = LocalConfiguration.current.locales[0]
    val calMonth = monthOfKey(calMonthKey)
    val calDay = LocalDate.ofEpochDay(calDayEpoch)
    val calWeekStart = remember(locale) { systemWeekStart(locale) }
    val onCalendar = route == "home" && tab == Tab.CALENDAR
    // Today's category chips and the page narrowed to the chosen one (v43).
    val repoProjects = remember(projects) { repoProjectIds(projects) }
    val workId = remember(categories) { workCategoryId(categories) }
    val chips = remember(mine, categories, repoProjects, workId) {
        mine?.let { todayChips(it, categories, repoProjects, workId) }.orEmpty()
    }
    val activeCat = if (chips.any { it.id == catFilter }) catFilter else 0L
    val shownMine = remember(mine, activeCat, repoProjects, workId) {
        mine?.let { filterMine(it, activeCat, repoProjects, workId) }
    }
    // A row stops naming the category it is filtered by.
    val rowCategories = remember(categories, activeCat) { categories.filterNot { it.id == activeCat } }
    val completedScope = parseCompletedRoute(route)
    val completedProject = completedScope?.projectId?.takeIf { it > 0 }?.let { id ->
        projects.firstOrNull { it.id == id }?.name ?: projectPage?.project?.takeIf { it.id == id }?.name
    }.orEmpty()
    val status = remember(syncStatusJson) {
        runCatching { PageJson.decodeFromString<SyncStatus>(syncStatusJson) }.getOrDefault(SyncStatus())
    }
    val fake = if (BuildConfig.DEBUG) remember(demoSyncState) { demoSyncState?.let { fakeSyncModel(it, resources) } } else null
    val shownStatus = fake?.status ?: liveStatus ?: status

    LaunchedEffect(syncStatusJson) {
        // A listener status is newer than the one a page read; let it win until the next read.
        liveStatus = null
    }

    fun push(next: String) {
        stack = stack + next
    }

    fun pop() {
        if (stack.size > 1) stack = stack.dropLast(1)
    }

    fun showNotice(text: String, undo: (() -> Unit)? = null) {
        notice = text
        noticeUndo = undo
        noticeId++
    }

    fun openTaskMenu(id: Long, page: TaskMenuPage = TaskMenuPage.Main) {
        menuPage = page
        scope.launch {
            val loaded = try {
                withContext(Dispatchers.IO) {
                    val task = api.task(id)
                    TaskMenuData(
                        task = task.task,
                        agent = task.latestAgent,
                        categories = api.categories(),
                        projects = api.projects(),
                        presets = api.remindPresets(),
                        milestones = task.task.projectId?.let { api.project(it).milestones }.orEmpty(),
                    )
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                Log.w("adhd-todo", "load failed", e)
                null
            }
            if (loaded != null) menuData = loaded
        }
    }

    fun deleteTask(task: Task) {
        val onTaskPage = route.startsWith("task:") &&
            route.substringAfter(':').substringBefore(':').toLongOrNull() == task.id
        scope.launch {
            val snapshot = try {
                withContext(Dispatchers.IO) { api.deleteTask(task.id) }
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                null
            }
            if (snapshot != null) {
                val message = resources.getString(R.string.menu_deleted, task.title)
                toast = UndoToast(message) {
                    scope.launch {
                        try {
                            withContext(Dispatchers.IO) { api.restoreTask(snapshot) }
                            tick++
                        } catch (e: CancellationException) {
                            throw e
                        } catch (e: Exception) {
                            Log.w("adhd-todo", "restore failed", e)
                        }
                    }
                }
                if (onTaskPage) pop()
                tick++
            }
        }
    }

    BackHandler(enabled = stack.size > 1) { pop() }
    // The system back leaves select mode first; so does moving to another page.
    BackHandler(enabled = selection != null) { selection = null }
    LaunchedEffect(route, tabName) { selection = null }
    LaunchedEffect(route, tabName) { assistant.dismiss() }
    // Which model would answer: read again with the sync status, and while it is not known yet.
    LaunchedEffect(syncStatusJson, revision) {
        while (true) {
            val read = withContext(Dispatchers.IO) { runCatching { api.modelState() }.getOrNull() }
            if (read != null) assistant.model = read
            delay(if (assistant.model.phase == ModelPhase.HIDDEN) 3000 else 60000)
        }
    }

    LaunchedEffect(noticeId) {
        if (notice != null) {
            delay(if (noticeUndo != null) 5000 else 1500)
            notice = null
            noticeUndo = null
        }
    }

    LaunchedEffect(toast) {
        if (toast == null) return@LaunchedEffect
        delay(6000)
        toast = null
    }

    // A row the composer just made keeps the iris 3 fill for 2 s.
    LaunchedEffect(highlightId) {
        if (highlightId == 0L) return@LaunchedEffect
        delay(2000)
        highlightId = 0L
    }

    LaunchedEffect(initialRoute) {
        if (initialRoute.isNullOrBlank()) return@LaunchedEffect
        val next = normalizeRoute(initialRoute)
        if (next == ROUTE_COMPOSE) {
            // A widget's + or Note something: Today, with the composer focused.
            tabName = Tab.MINE.name
            stack = listOf("home")
            composeFocus++
        } else if (next.startsWith("tab:")) {
            // route=projects opens the Projects tab, not Mine with the projects route on top.
            tabName = runCatching { Tab.valueOf(next.removePrefix("tab:").uppercase()) }
                .getOrDefault(Tab.MINE).name
            stack = listOf("home")
        } else if (next.isNotBlank()) {
            stack = if (next == "home") listOf("home") else listOf("home", next)
        }
        onInitialRouteConsumed()
    }

    LaunchedEffect(route, revision, tick, tab, calMonthKey, calWeekStart) {
        when {
            route == "home" -> when (tab) {
                Tab.MINE -> {
                    val loaded = try {
                        withContext(Dispatchers.IO) {
                            val page = api.mine()
                            Triple(page, api.agentInfo(page), runCatching { api.categories() }.getOrNull())
                        }
                    } catch (e: CancellationException) {
                        throw e
                    } catch (_: Exception) {
                        null
                    }
                    if (loaded != null) {
                        mine = loaded.first
                        mineAgents = loaded.second
                        loaded.third?.let { categories = it }
                    }
                }
                Tab.AGENTS -> {
                    val loaded = try {
                        withContext(Dispatchers.IO) {
                            val page = api.agents()
                            page to api.info(page.working.map { it.id } + page.done.map { it.id })
                        }
                    } catch (e: CancellationException) {
                        throw e
                    } catch (_: Exception) {
                        null
                    }
                    if (loaded != null) {
                        agents = loaded.first
                        agentsInfo = loaded.second
                    }
                }
                Tab.PROJECTS -> {
                    val loaded = try {
                        withContext(Dispatchers.IO) {
                            api.projectEntries() to archivedEntries(api.archivedProjects()) { id ->
                                runCatching { api.project(id).tasks.size }.getOrDefault(0)
                            }
                        }
                    } catch (e: CancellationException) {
                        throw e
                    } catch (_: Exception) {
                        null
                    }
                    if (loaded != null) {
                        projectEntries = loaded.first
                        archivedProjects = loaded.second
                    }
                }
                Tab.CALENDAR -> {
                    val loaded = try {
                        withContext(Dispatchers.IO) {
                            api.calendar(calMonth.year, calMonth.monthValue, coreWeekStart(calWeekStart)) to
                                runCatching { api.categories() }.getOrNull()
                        }
                    } catch (e: CancellationException) {
                        throw e
                    } catch (_: Exception) {
                        null
                    }
                    if (loaded != null) {
                        calendarPage = loaded.first
                        loaded.second?.let { categories = it }
                    }
                }
            }
            route.startsWith("project:") -> {
                val id = route.substringAfter(':').toLongOrNull() ?: return@LaunchedEffect
                val loaded = try {
                    withContext(Dispatchers.IO) { api.project(id) }
                } catch (e: CancellationException) {
                    throw e
                } catch (_: Exception) {
                    null
                }
                if (loaded != null) projectPage = loaded
            }
            route.startsWith("task:") -> {
                val id = route.substringAfter(':').substringBefore(':').toLongOrNull() ?: return@LaunchedEffect
                val loaded = try {
                    withContext(Dispatchers.IO) {
                        Triple(api.task(id), api.projects(), api.categories())
                    }
                } catch (e: CancellationException) {
                    throw e
                } catch (e: Exception) {
                    Log.w("adhd-todo", "load failed", e)
                    null
                }
                if (loaded != null) {
                    taskPage = loaded.first
                    projects = loaded.second
                    categories = loaded.third
                    taskMilestones = withContext(Dispatchers.IO) {
                        loaded.first.task.projectId
                            ?.let { runCatching { api.project(it).milestones }.getOrNull() }
                            .orEmpty()
                    }
                }
            }
            route == ROUTE_SETTINGS || route == ROUTE_CATEGORIES -> {
                val loaded = try {
                    withContext(Dispatchers.IO) { loadSettingsData(api) }
                } catch (e: CancellationException) {
                    throw e
                } catch (_: Exception) {
                    null
                }
                if (loaded != null) {
                    settingsData = loaded
                    account = loaded.account
                    devices = loaded.devices
                    loaded.status?.let { liveStatus = it }
                }
            }
            route == "sync" -> {
                val loaded = try {
                    withContext(Dispatchers.IO) {
                        Triple(
                            api.account(),
                            runCatching { api.devices() }.getOrDefault(emptyList()),
                            runCatching { api.syncStatus() }.getOrNull(),
                        )
                    }
                } catch (e: CancellationException) {
                    throw e
                } catch (_: Exception) {
                    null
                }
                if (loaded != null) {
                    account = loaded.first
                    devices = loaded.second
                    loaded.third?.let { liveStatus = it }
                }
            }
        }
    }

    // Leaving Today for another tab clears the category chip; it is not kept.
    LaunchedEffect(tab) { if (tab != Tab.MINE) catFilter = 0L }

    // The chips and a Completed page's back bar need the projects (repositories, names).
    LaunchedEffect(route, tab, revision, tick) {
        if ((route == "home" && tab == Tab.MINE) || route.startsWith(CompletedPrefix)) {
            val loaded = try {
                withContext(Dispatchers.IO) { api.projects() }
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                null
            }
            if (loaded != null) projects = loaded
        }
    }
    SummaryLoader(summary, api, route == "summary", revision, tick)

    // The action is the trailing lambda; name onError when passing it, since both are lambdas of
    // no arguments and swapping them still compiles.
    fun perform(onError: (() -> Unit)? = null, action: suspend () -> Unit) {
        scope.launch {
            try {
                withContext(Dispatchers.IO) { action() }
                tick++
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                android.util.Log.w("adhd-todo", "action failed", e)
                onError?.invoke()
            }
        }
    }

    val projectActions = ProjectActions(
        createProject = { name, parentId ->
            val made = withContext(Dispatchers.IO) { api.createProject(name, parentId) }
            tick++
            push("project:${made.id}")
        },
        renameProject = { id, name -> withContext(Dispatchers.IO) { api.renameProject(id, name) }; tick++ },
        archiveProject = { id -> withContext(Dispatchers.IO) { api.archiveProject(id) }; tick++ },
        unarchiveProject = { id -> withContext(Dispatchers.IO) { api.unarchiveProject(id) }; tick++ },
        deleteProject = { id -> withContext(Dispatchers.IO) { api.deleteProject(id) }; tick++ },
        createMilestone = { id, title, due -> withContext(Dispatchers.IO) { api.createMilestone(id, title, due) }; tick++ },
        updateMilestone = { id, patch -> withContext(Dispatchers.IO) { api.updateMilestone(id, patch) }; tick++ },
    )
    val categoryActions = CategoryActions(
        create = { input -> withContext(Dispatchers.IO) { api.createCategory(input) }.also { tick++ } },
        update = { id, input -> withContext(Dispatchers.IO) { api.updateCategory(id, input) }.also { tick++ } },
        delete = { id -> withContext(Dispatchers.IO) { api.deleteCategory(id) }.also { tick++ } },
        restore = { snapshot -> withContext(Dispatchers.IO) { api.restoreCategory(snapshot) }; tick++ },
        move = { id, index -> withContext(Dispatchers.IO) { api.moveCategory(id, index.toLong()) }; tick++ },
        icons = { query -> withContext(Dispatchers.IO) { api.categoryIcons(query) } },
    )

    fun shiftCalendar(by: Int) {
        val (month, day) = shiftMonth(calMonth, calDay, by)
        calMonthKey = month.key()
        calDayEpoch = day.toEpochDay()
    }

    fun calendarToday() {
        calMonthKey = YearMonth.now().key()
        calDayEpoch = LocalDate.now().toEpochDay()
    }

    fun startSync() {
        scope.launch {
            runCatching { withContext(Dispatchers.IO) { api.startSync() } }
        }
    }

    // What the parser read out of the text (v43); tapping the preview line ignores it for this text.
    val quickParse = rememberQuickParse(api, draft, enabled = !route.startsWith("task:"))
    var ignoredQuick by remember { mutableStateOf<String?>(null) }
    val activeQuick = quickParse?.takeIf { it.readAny() && ignoredQuick != draft }

    val submitLocal: (QuietNote?) -> Unit = { note ->
        assistant.dismiss()
        val title = draft.trim()
        val projectId = if (route.startsWith("project:")) route.substringAfter(':').toLongOrNull() else null
        val quick = activeQuick
        // A task typed while a chip narrows Today gets that category (the desktop's rule).
        val typedCat = activeCat.takeIf { it > 0 && route == "home" && tab == Tab.MINE }
        // Under the Calendar a task is due on the selected day, at 18:00 unless its text says otherwise (v44).
        val dayDue = if (onCalendar) calendarDue(calDay, ZoneId.systemDefault()) else null
        if (title.isNotEmpty() && !sending) {
            sending = true
            scope.launch {
                val madeAll = try {
                    withContext(Dispatchers.IO) {
                        if (quick != null) {
                            createQuickDrafts(api, quick, projectId, typedCat, dayDue)
                        } else {
                            listOf(api.createTask(title, projectId, dueAt = dayDue, categoryId = typedCat))
                        }
                    }
                } catch (e: CancellationException) {
                    throw e
                } catch (_: Exception) {
                    null
                }
                val made = madeAll?.firstOrNull()
                sending = false
                if (made != null) {
                    draft = ""
                    // An undated add opens the No date fold, where the new row shows for 2 s.
                    if (onCalendar) {
                        highlightId = made
                    } else if (projectId == null && quick == null) {
                        noDateOpen = true
                        highlightId = made
                    }
                    if (note != null) {
                        assistant.card = localCard(
                            resources, words, quick, if (quick != null) quick.drafts.map { it.title } else listOf(title),
                            madeAll.orEmpty(), note,
                        )
                    } else showNotice(
                        if (quick != null && (!onCalendar || quickReadsDate(quick))) {
                            quickReceipt(resources, words, quick)
                        } else if (onCalendar) {
                            resources.getString(R.string.calendar_added, calendarShortDay(calDay, locale))
                        } else if (projectId != null) {
                            resources.getString(R.string.added_to_project, projectPage?.project?.name.orEmpty())
                        } else {
                            resources.getString(R.string.added_to_inbox)
                        },
                        undo = {
                            highlightId = 0L
                            perform { madeAll.orEmpty().forEach { api.deleteTask(it) } }
                            notice = null
                            noticeUndo = null
                        },
                    )
                    tick++
                } else {
                    showNotice(resources.getString(R.string.add_failed))
                }
            }
        }
    }

    // With the official model there the box asks it (v44); it falls back to the local rules above.
    val onTaskPage = route.startsWith("task:")
    val askModel: (String) -> Unit = { text ->
        if (text.isNotEmpty() && !sending && !assistant.thinking) {
            val place = assistantPlace(
                route, tab, calDay, activeCat.takeIf { it > 0 && route == "home" && tab == Tab.MINE } ?: 0L,
                ZoneId.systemDefault(),
            )
            scope.launch {
                when (val outcome = assistant.runTurn(api, resources, words, text, place)) {
                    is TurnOutcome.Answer -> {
                        if (text == draft.trim()) draft = ""
                        val added = assistant.card?.newTaskId ?: 0L
                        if (added != 0L) {
                            highlightId = added
                            // An undated task opens the No date fold, where the new row shows.
                            if (!onCalendar && !route.startsWith("project:") &&
                                outcome.reply.receipts.any { it.taskId == added && it.detail.kind == "added" }
                            ) {
                                noDateOpen = true
                            }
                        }
                        tick++
                    }
                    is TurnOutcome.Fallback -> submitLocal(outcome.note)
                    else -> Unit
                }
                delay(1500)
                runCatching { withContext(Dispatchers.IO) { api.modelState() } }.getOrNull()?.let { assistant.model = it }
            }
        }
    }
    val submit: () -> Unit = {
        if (assistant.asks(onTaskPage)) askModel(draft.trim()) else submitLocal(if (onTaskPage) null else quietForModel(assistant.model))
    }
    val undoCard: () -> Unit = {
        val undo = assistant.card?.undo
        assistant.dismiss()
        highlightId = 0L
        when (undo) {
            is CardUndo.Receipts -> perform { undo.ids.forEach { api.undo(it) } }
            is CardUndo.Tasks -> perform { undo.ids.forEach { api.deleteTask(it) } }
            null -> Unit
        }
    }

    fun runBulk(action: BulkAction) {
        val ids = selection.orEmpty().toList()
        if (ids.isEmpty()) return
        scope.launch {
            val outcome = try {
                withContext(Dispatchers.IO) { runBulkAction(api, action, ids, resources, words) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                Log.w("adhd-todo", "bulk action failed", e)
                null
            }
            selection = null
            if (outcome == null) {
                showNotice(resources.getString(R.string.bulk_failed))
            } else {
                val undo = outcome.undo
                if (undo != null) toast = UndoToast(outcome.message) { perform { undo() } } else showNotice(outcome.message)
                tick++
            }
        }
    }

    fun swipe(task: SwipedTask, action: SwipeAction) {
        scope.launch {
            val zone = ZoneId.systemDefault()
            val outcome = try {
                withContext(Dispatchers.IO) {
                    runSwipe(api, resources, task, action, java.time.LocalDate.now(zone), zone)
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                Log.w("adhd-todo", "swipe failed", e)
                null
            }
            if (outcome != null) {
                toast = UndoToast(outcome.message) { perform { outcome.undo() } }
                tick++
            }
        }
    }

    val sendReply: () -> Unit = {
        val text = draft.trim()
        val id = route.substringAfter(':').substringBefore(':').toLongOrNull()
        if (text.isNotEmpty() && id != null && !sending) {
            sending = true
            scope.launch {
                val ok = try {
                    withContext(Dispatchers.IO) { api.reply(id, text) }
                    true
                } catch (e: CancellationException) {
                    throw e
                } catch (_: Exception) {
                    false
                }
                sending = false
                if (ok) {
                    draft = ""
                    showNotice(resources.getString(R.string.reply_sent))
                    tick++
                } else {
                    showNotice(resources.getString(R.string.reply_failed))
                }
            }
        }
    }

    val openUrl: (String) -> Unit = { url ->
        if (url.isNotBlank()) {
            try {
                context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
            } catch (_: Exception) {
                // A link Android cannot open is not worth a dialog.
            }
        }
    }

    // Google sign-in opens in a Custom Tab (toolbar in the surface colour), and
    // falls back to the default browser where no browser supports Custom Tabs.
    val openCustomTab: (String) -> Unit = { url ->
        if (url.isNotBlank()) {
            try {
                CustomTabsIntent.Builder()
                    .setDefaultColorSchemeParams(
                        CustomTabColorSchemeParams.Builder()
                            .setToolbarColor(tokens.bg.toArgb())
                            .build(),
                    )
                    .build()
                    .launchUrl(context, Uri.parse(url))
            } catch (_: Exception) {
                openUrl(url)
            }
        }
    }

    // The Google row of the signed-in card: asked once the screen opens with a linked account.
    val signedInHere = shownStatus.signedIn && shownStatus.linked
    LaunchedEffect(route == "sync" && signedInHere) {
        if (route == "sync" && signedInHere) {
            googleRow = withContext(Dispatchers.IO) {
                val available = runCatching { api.googleLinkAvailable() }.getOrDefault(false)
                val linked = available && runCatching { api.googleLinked() }.getOrDefault(false)
                GoogleLinkRow(available = available, linked = linked)
            }
        }
    }

    // Continue with Google shows only where the server lists a Google provider.
    LaunchedEffect(route == "sync") {
        if (route == "sync") {
            googleAvailable = withContext(Dispatchers.IO) { runCatching { api.googleAvailable() }.getOrDefault(false) }
        }
    }

    val blocked = syncCloudBlocked(
        linked = shownStatus.linked,
        signedIn = shownStatus.signedIn,
        state = shownStatus.state,
        failedSinceMillis = shownStatus.failedSince?.let { runCatching { Instant.parse(it).toEpochMilli() }.getOrNull() },
        nowMillis = System.currentTimeMillis(),
    )

    fun openSync() {
        if (route != "sync") push("sync")
    }

    // Pull to sync (v43 P2): the gap stays open while the runner reports syncing, at least 800 ms.
    var pullSyncing by remember { mutableStateOf(false) }
    val syncState = rememberUpdatedState(shownStatus.state)
    fun pullSync() {
        if (pullSyncing) return
        pullSyncing = true
        scope.launch {
            val started = System.currentTimeMillis()
            val failed = try {
                withContext(Dispatchers.IO) { api.syncNow() }
                false
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                Log.w("adhd-todo", "pull to sync failed", e)
                true
            }
            if (!failed) {
                // The runner takes the nudge a moment later; wait for it to start, then to end.
                withTimeoutOrNull(1500) { snapshotFlow { syncState.value }.first { it == "syncing" } }
                withTimeoutOrNull(15000) { snapshotFlow { syncState.value }.first { it != "syncing" } }
            }
            delay((800 - (System.currentTimeMillis() - started)).coerceAtLeast(0))
            pullSyncing = false
            if (failed) showNotice(resources.getString(R.string.sync_now_failed))
            tick++
        }
    }
    val pullRoute = route == "home" || route.startsWith("project:") || completedScope != null

    val onToday = route == "home" && tab == Tab.MINE
    val title = when {
        onToday -> words.headerDate()
        route == "sync" -> stringResource(R.string.sync_title)
        route.startsWith("project:") -> stringResource(R.string.title_projects)
        route.startsWith("task:") -> when (tab) {
            Tab.AGENTS -> stringResource(R.string.title_agents)
            Tab.PROJECTS -> stringResource(R.string.title_projects)
            else -> stringResource(R.string.title_mine)
        }
        tab == Tab.MINE -> stringResource(R.string.title_mine)
        tab == Tab.AGENTS -> stringResource(R.string.title_agents)
        tab == Tab.CALENDAR -> stringResource(R.string.title_calendar)
        else -> stringResource(R.string.title_projects)
    }

    val subtitle = when {
        route != "home" -> ""
        tab == Tab.MINE -> ""
        tab == Tab.AGENTS -> {
            val working = agents?.working?.size ?: 0
            if (working > 0) resources.getString(R.string.agents_working, working) else ""
        }
        else -> ""
    }

    val composerVisible = route == "home" || route.startsWith("project:") || route.startsWith("task:")
    val composerPlaceholder = when {
        route.startsWith("task:") -> {
            val agent = taskPage?.latestAgent.orEmpty()
            if (agent.isNotEmpty()) {
                resources.getString(R.string.composer_reply_to, words.agent(agent))
            } else {
                resources.getString(R.string.composer_add_note)
            }
        }
        route.startsWith("project:") -> resources.getString(
            R.string.composer_add_to_project,
            projectPage?.project?.name.orEmpty(),
        )
        tab == Tab.CALENDAR -> resources.getString(R.string.calendar_add_to, calendarShortDay(calDay, locale))
        tab == Tab.MINE -> stringResource(R.string.composer_placeholder_mine)
        tab == Tab.AGENTS -> stringResource(R.string.composer_placeholder_agents)
        else -> stringResource(R.string.composer_placeholder_inbox)
    }

    val selectionUi = selection?.let { ticked ->
        SelectionUi(
            ids = ticked,
            order = selectOrder,
            toggle = { id -> selection = ticked.toggled(id) },
            report = { selectOrder = it },
        )
    }
    CompositionLocalProvider(LocalSelection provides selectionUi) {
    Column(Modifier.fillMaxSize().background(tokens.bg)) {
        Column(Modifier.weight(1f).statusBarsPadding().imePadding()) {
            if (selection != null) {
                SelectHeader(
                    count = selection.orEmpty().size,
                    onSelectAll = { selection = selectOrder.filter { it != 0L }.toSet() },
                    onDone = { selection = null },
                )
            } else if (route == "search") {
                // The page draws its own field and Cancel.
            } else if (route == "home") {
                if (tab == Tab.CALENDAR) {
                    CalendarHeader(
                        title = calendarTitle(calMonth, locale),
                        syncBlocked = blocked,
                        onSync = { openSync() },
                        onPrevious = { shiftCalendar(-1) },
                        onToday = { calendarToday() },
                        onNext = { shiftCalendar(1) },
                    )
                } else {
                    PageHeader(
                        title = title,
                        subtitle = subtitle,
                        syncBlocked = blocked,
                        onSync = { openSync() },
                        start = if (onToday) 28.dp else Dims.pagePadding,
                        onSearch = { searchQuery = ""; push("search") },
                        actions = {
                            if (tab == Tab.PROJECTS) {
                                HeaderIconButton("plus", stringResource(R.string.manage_new_project), { newProject = true })
                                HeaderIconButton("gear", stringResource(R.string.settings_title), { push(ROUTE_SETTINGS) })
                            }
                        },
                    )
                }
                if (update.bannerShown) UpdateBanner(update, onUpdate = onUpdate, onDismiss = onDismissUpdate)
                if (onToday && chips.isNotEmpty()) {
                    CategoryChips(chips = chips, selected = activeCat, words = words, onSelect = { catFilter = it })
                }
            } else if (completedScope != null) {
                BackBar(
                    title = when {
                        completedScope.projectId > 0 -> completedProject
                        completedScope.agents -> stringResource(R.string.title_agents)
                        else -> stringResource(R.string.title_mine)
                    },
                    onBack = { pop() },
                )
            } else if (route == "sync") {
                BackBar(title = stringResource(R.string.sync_title), onBack = { pop() })
            } else if (isSettingsRoute(route)) {
                BackBar(
                    title = stringResource(if (route == ROUTE_SETTINGS) R.string.title_projects else R.string.settings_title),
                    onBack = { pop() },
                )
            } else if (route == "summary") {
                BackBar(title = title, onBack = { pop() })
            } else if (route.startsWith("project:")) {
                BackBar(
                    title = stringResource(R.string.title_projects),
                    onBack = { pop() },
                    right = { SyncCloud(blocked = blocked, onSync = { openSync() }) },
                )
            } else {
                val taskId = route.substringAfter(':').substringBefore(':').toLongOrNull() ?: 0
                TaskBackBar(
                    title = title,
                    number = taskId,
                    onBack = { pop() },
                    onMenu = { openTaskMenu(taskId, TaskMenuPage.Main) },
                )
            }

            PullToSync(
                enabled = pullRoute && selection == null && pullSyncs(shownStatus),
                syncing = pullSyncing,
                onSync = { pullSync() },
                modifier = Modifier.weight(1f),
            ) {
            Box(Modifier.fillMaxSize().dismissOnTouch(assistant.card != null) { assistant.dismiss() }) {
                when {
                    route == "sync" -> {
                        val model = fake ?: SyncModel(
                            phase = syncPhaseOf(shownStatus, signingIn, choosing, otherChoosing),
                            status = shownStatus,
                            account = account ?: Account(),
                            devices = devices,
                            choose = choose,
                            flow = flow,
                            googleAvailable = googleAvailable,
                            otherAccount = otherAccount,
                            googleLink = googleRow,
                        )
                        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(horizontal = Dims.pagePadding)) {
                            syncMessage?.let {
                                androidx.compose.material3.Text(
                                    it,
                                    style = xyz.sriai.adhdtodo.ui.theme.LocalAppType.current.meta,
                                    color = tokens.statusBlocked,
                                    modifier = Modifier.padding(top = 8.dp),
                                )
                            }
                            val runSignIn: (() -> SignInState) -> Unit = { call ->
                                if (!signingIn) {
                                    signingIn = true
                                    choosing = false
                                    otherChoosing = false
                                    syncMessage = null
                                    flow = flow.copy(failure = null, failureText = null)
                                    scope.launch {
                                        var failure: Exception? = null
                                        val result = try {
                                            withContext(Dispatchers.IO) { call() }
                                        } catch (e: CancellationException) {
                                            throw e
                                        } catch (e: Exception) {
                                            failure = e
                                            null
                                        }
                                        if (failure != null) {
                                            val state = withContext(Dispatchers.IO) {
                                                runCatching { api.codeState() }.getOrNull()
                                            }
                                            flow = flow.failed(state, failure.message)
                                        }
                                        signingIn = false
                                        if (failure != null) Log.w("adhd-todo", "sign-in failed", failure)
                                        when (result?.state) {
                                            "choose" -> {
                                                choose = checkNotNull(result)
                                                choosing = true
                                                flow = SignInFlow()
                                            }
                                            "other_account" -> {
                                                // Nothing synced; the sheet asks which account the phone belongs to.
                                                otherAccount = checkNotNull(result)
                                                otherChoosing = true
                                                flow = SignInFlow()
                                            }
                                            "linked" -> {
                                                flow = SignInFlow()
                                                startSync()
                                            }
                                            "verify" -> {
                                                // The password was right but the email never verified: a code was mailed.
                                                val state = withContext(Dispatchers.IO) {
                                                    runCatching { api.codeState() }.getOrNull()
                                                }
                                                flow = flow.afterVerifyAnswer(state)
                                            }
                                            null -> if (failure == null) {
                                                syncMessage = resources.getString(R.string.sync_sign_in_failed)
                                            }
                                            else -> Unit
                                        }
                                        tick++
                                    }
                                }
                            }
                            SyncScreen(
                                model = model,
                                words = words,
                                update = update,
                                onCheckUpdate = onCheckUpdate,
                                onUpdate = onUpdate,
                                serverEditing = serverEditing,
                                serverDraft = serverDraft,
                                serverError = serverError,
                                serverSaving = serverSaving,
                                onSignIn = { login, password ->
                                    runSignIn { api.signIn(login, password, Build.MODEL) }
                                },
                                signInActions = SignInActions(
                                    onSignUp = { email, password ->
                                        if (!signingIn && !flow.creating && !flow.sending) {
                                            flow = flow.copy(email = email, failure = null, failureText = null, creating = true)
                                            scope.launch {
                                                val error = withContext(Dispatchers.IO) {
                                                    runCatching { api.signUp(email, password, syncLang(java.util.Locale.getDefault())) }
                                                        .exceptionOrNull()
                                                }
                                                if (error is CancellationException) throw error
                                                val state = withContext(Dispatchers.IO) {
                                                    runCatching { api.codeState() }.getOrNull()
                                                }
                                                if (error != null) Log.w("adhd-todo", "sign up failed", error)
                                                flow = flow.afterSignUp(error?.let { it.message ?: "" }, state)
                                            }
                                        }
                                    },
                                    onSendResetCode = { email ->
                                        if (!signingIn && !flow.creating && !flow.sending) {
                                            flow = flow.copy(email = email, failure = null, failureText = null, sending = true)
                                            scope.launch {
                                                val error = withContext(Dispatchers.IO) {
                                                    runCatching { api.requestResetCode(email) }.exceptionOrNull()
                                                }
                                                if (error is CancellationException) throw error
                                                val state = withContext(Dispatchers.IO) {
                                                    runCatching { api.codeState() }.getOrNull()
                                                }
                                                if (error != null) Log.w("adhd-todo", "request reset code failed", error)
                                                flow = flow.afterResetRequest(error?.let { it.message ?: "" }, state)
                                            }
                                        }
                                    },
                                    onSubmitCode = { code -> runSignIn { api.signInWithCode(code, Build.MODEL) } },
                                    onResend = {
                                        scope.launch {
                                            val error = withContext(Dispatchers.IO) {
                                                runCatching { api.resendCode() }.exceptionOrNull()
                                            }
                                            if (error is CancellationException) throw error
                                            val state = withContext(Dispatchers.IO) {
                                                runCatching { api.codeState() }.getOrNull()
                                            }
                                            flow = if (error == null) {
                                                flow.copy(
                                                    resendIn = state?.resendIn ?: 30,
                                                    failure = null,
                                                    failureText = null,
                                                    attempt = flow.attempt + 1,
                                                )
                                            } else {
                                                flow.failed(state, error.message)
                                            }
                                        }
                                    },
                                    onResetPassword = { code, password ->
                                        runSignIn { api.resetPassword(code, password, Build.MODEL) }
                                    },
                                    onOpenUrl = { url -> openCustomTab(url) },
                                    onStep = { step ->
                                        flow = flow.copy(step = step, failure = null, failureText = null, attempt = flow.attempt + 1)
                                    },
                                    // The page opens in a Custom Tab; the core keeps the wait until it answers.
                                    onGoogle = {
                                        runSignIn {
                                            val url = api.googleStart()
                                            Handler(Looper.getMainLooper()).post { openCustomTab(url) }
                                            api.googleFinish(Build.MODEL)
                                        }
                                    },
                                    onCancelGoogle = { scope.launch(Dispatchers.IO) { api.googleCancel() } },
                                ),
                                onSignOut = {
                                    flow = SignInFlow()
                                    choosing = false
                                    otherChoosing = false
                                    signingIn = false
                                    perform { api.signOut() }
                                },
                                onSyncNow = {
                                    syncMessage = null
                                    perform(onError = { syncMessage = resources.getString(R.string.sync_now_failed) }) { api.syncNow() }
                                },
                                onRetry = {
                                    syncMessage = null
                                    perform(onError = { syncMessage = resources.getString(R.string.sync_now_failed) }) { api.syncNow() }
                                },
                                onMerge = {
                                    choosing = false
                                    syncMessage = null
                                    perform(onError = { syncMessage = resources.getString(R.string.sync_link_failed) }) {
                                        api.linkMerge(Build.MODEL)
                                        api.startSync()
                                    }
                                },
                                onReplace = {
                                    choosing = false
                                    syncMessage = null
                                    perform(onError = { syncMessage = resources.getString(R.string.sync_link_failed) }) {
                                        api.linkReplace(Build.MODEL)
                                        api.startSync()
                                    }
                                },
                                onCancelChoose = {
                                    choosing = false
                                    otherChoosing = false
                                    perform { api.signOut() }
                                },
                                onUseOldAccount = { previous ->
                                    otherChoosing = false
                                    signingIn = false
                                    flow = SignInFlow(email = previous)
                                    perform { api.signOut() }
                                },
                                onMoveToAccount = { runSignIn { api.moveToAccount(Build.MODEL) } },
                                onGoogleLink = {
                                    if (!googleRow.busy) {
                                        googleRow = googleRow.copy(busy = true, message = null)
                                        scope.launch {
                                            // null is success; else the core's error kind.
                                            val kind = withContext(Dispatchers.IO) {
                                                try {
                                                    val url = api.googleLinkStart()
                                                    Handler(Looper.getMainLooper()).post { openCustomTab(url) }
                                                    api.googleLinkFinish()
                                                    null
                                                } catch (e: CancellationException) {
                                                    throw e
                                                } catch (e: Exception) {
                                                    Log.w("adhd-todo", "google link failed", e)
                                                    e.message?.trim().orEmpty().ifEmpty { "failed" }
                                                }
                                            }
                                            googleRow = if (kind == null) {
                                                googleRow.copy(busy = false, linked = true, message = null)
                                            } else {
                                                googleRow.copy(busy = false, message = googleLinkFailureText(kind))
                                            }
                                        }
                                    }
                                },
                                onCancelGoogleLink = { scope.launch(Dispatchers.IO) { api.googleLinkCancel() } },
                                onStartServerEdit = {
                                    serverEditing = true
                                    serverDraft = shownStatus.server.takeUnless { isOfficialSyncServer(it) }.orEmpty()
                                    serverError = false
                                },
                                onServerDraft = { serverDraft = it; serverError = false },
                                onSaveServer = {
                                    if (!serverSaving) {
                                        serverSaving = true
                                        serverError = false
                                        scope.launch {
                                            val ok = try {
                                                withContext(Dispatchers.IO) { api.syncSetServer(syncServerToSave(serverDraft)) }
                                                true
                                            } catch (e: CancellationException) {
                                                throw e
                                            } catch (_: Exception) {
                                                false
                                            }
                                            serverSaving = false
                                            if (ok) {
                                                serverEditing = false
                                                tick++
                                            } else {
                                                serverError = true
                                            }
                                        }
                                    }
                                },
                                onCancelServerEdit = { serverEditing = false; serverError = false },
                            )
                        }
                    }
                    route == "search" -> SearchRoute(
                        api = api,
                        words = words,
                        revision = revision,
                        tick = tick,
                        query = searchQuery,
                        onQueryChange = { searchQuery = it },
                        onCancel = { pop() },
                        onOpenTask = { id -> push(taskRoute(id, 0)) },
                        onLongPress = { id -> openTaskMenu(id) },
                        onToggle = { task ->
                            when {
                                task.status == "done" || task.status == "cancelled" ->
                                    perform { api.updateTask(task.id, TaskPatch(status = "todo")) }
                                task.source == "user" -> perform { api.complete(task.id) }
                                else -> push(taskRoute(task.id, 0))
                            }
                        },
                    )
                    completedScope != null -> CompletedRoute(
                        api = api,
                        scope = completedScope,
                        scopeName = when {
                            completedScope.projectId > 0 -> completedProject
                            completedScope.agents -> stringResource(R.string.title_agents)
                            else -> ""
                        },
                        words = words,
                        revision = revision,
                        tick = tick,
                        onOpenTask = { id -> push(taskRoute(id, 0)) },
                        onLongPress = { id -> openTaskMenu(id) },
                        onReopen = { id -> perform { api.updateTask(id, TaskPatch(status = "todo")) } },
                    )
                    route == ROUTE_SETTINGS -> SettingsRoute(
                        data = settingsData.copy(model = assistant.model),
                        status = shownStatus,
                        update = update,
                        onOpenSync = { openSync() },
                        onNotifications = { push(ROUTE_NOTIFICATIONS) },
                        onCategories = { push(ROUTE_CATEGORIES) },
                        onCheckUpdate = onCheckUpdate,
                        onUpdate = onUpdate,
                        onOpenUrl = openUrl,
                    )
                    route == ROUTE_NOTIFICATIONS -> NotificationSettingsRoute(
                        onChanged = { scope.launch(Dispatchers.IO) { runCatching { Reminders.schedule(context, core) } } },
                    )
                    route == ROUTE_CATEGORIES -> CategoriesScreen(
                        categories = settingsData.categories,
                        actions = categoryActions,
                        onChanged = { tick++ },
                    )
                    route == "summary" -> SummaryScreen(
                        holder = summary,
                        words = words,
                        onOpenTask = { id -> push(taskRoute(id, 0)) },
                        onShare = { scope.launch { shareSummary(context, api, summary.week, summary.back) } },
                    )
                    route.startsWith("project:") -> ProjectScreen(
                        page = projectPage,
                        words = words,
                        onOpenTask = { id -> push(taskRoute(id, route.substringAfter(':').toLongOrNull() ?: 0)) },
                        onLongPress = { id -> openTaskMenu(id) },
                        onShowAllCompleted = { agentsOnly ->
                            val id = route.substringAfter(':').toLongOrNull() ?: 0
                            push(completedRoute(CompletedScope(projectId = id, agents = agentsOnly)))
                        },
                        actions = projectActions,
                        onSwipe = { task, action -> swipe(SwipedTask(task.id, task.title, task.dueAt), action) },
                    )
                    route.startsWith("task:") -> TaskScreen(
                        page = taskPage,
                        projects = projects,
                        words = words,
                        onSetPriority = { priority ->
                            val id = route.substringAfter(':').substringBefore(':').toLongOrNull()
                            if (id != null) perform { api.updateTask(id, TaskPatch(priority = priority)) }
                        },
                        onOpenUrl = openUrl,
                        onOpenTask = { id -> push(taskRoute(id, route.substringAfter(':').substringBefore(':').toLongOrNull() ?: 0)) },
                        categories = categories,
                        onOpenMenu = { page ->
                            val id = route.substringAfter(':').substringBefore(':').toLongOrNull()
                            if (id != null) openTaskMenu(id, page)
                        },
                        milestones = taskMilestones,
                        onSaveNotes = { notes ->
                            val id = route.substringAfter(':').substringBefore(':').toLongOrNull()
                            if (id != null) perform { api.updateTask(id, TaskPatch(notes = notes)) }
                        },
                        onAddSubtask = { title ->
                            val parent = taskPage?.task
                            if (parent != null) {
                                perform { api.createTask(title, projectId = parent.projectId, parentId = parent.id) }
                            }
                        },
                    )
                    else -> when (tab) {
                        Tab.MINE -> MineScreen(
                            page = shownMine,
                            agents = mineAgents,
                            words = words,
                            onAccept = { id -> perform { api.accept(id) } },
                            onComplete = { id -> perform { api.complete(id) } },
                            onAnswer = { replyTo = it },
                            onOpenUrl = openUrl,
                            onOpenTask = { id -> push(taskRoute(id, 0)) },
                            onHowToConnect = { connect = true },
                            // The phone has no Summary; the Agents tab lists what agents finished today.
                            onViewAll = null,
                            onSummary = { summary.open(); push("summary") },
                            onLongPress = { id -> openTaskMenu(id) },
                            categories = rowCategories,
                            noDateOpen = noDateOpen,
                            onToggleNoDate = { noDateOpen = !noDateOpen },
                            highlightId = highlightId,
                            onShowAll = { push(completedRoute(CompletedScope())) },
                            onSwipe = { item, action -> swipe(SwipedTask(item.id, item.title, item.dueAt), action) },
                        )
                        Tab.AGENTS -> AgentsScreen(
                            page = agents,
                            agents = agentsInfo,
                            words = words,
                            onOpenTask = { id -> push(taskRoute(id, 0)) },
                            onHowToConnect = { connect = true },
                            onLongPress = { id -> openTaskMenu(id) },
                            onShowAll = { push(completedRoute(CompletedScope(agents = true))) },
                        )
                        Tab.CALENDAR -> CalendarScreen(
                            page = calendarPage,
                            month = calMonth,
                            selected = calDay,
                            today = LocalDate.now(),
                            weekStart = calWeekStart,
                            words = words,
                            categories = categories,
                            onSelect = { calDayEpoch = it.toEpochDay() },
                            onShift = { shiftCalendar(it) },
                            onOpenTask = { id -> push(taskRoute(id, 0)) },
                            onLongPress = { id -> openTaskMenu(id) },
                            onComplete = { id -> perform { api.complete(id) } },
                            highlightId = highlightId,
                        )
                        Tab.PROJECTS -> ProjectsScreen(
                            entries = projectEntries,
                            onOpenProject = { id -> push("project:$id") },
                            onHowToConnect = { connect = true },
                            archived = archivedProjects,
                            actions = projectActions,
                            onOpenUrl = openUrl,
                        )
                    }
                }
            }
            }

            toast?.let { current ->
                UndoBar(message = current.message, onUndo = { toast = null; current.undo() })
            }

            if (composerVisible && selection == null) {
                val placeholder = if (assistant.card?.question != null) {
                    stringResource(R.string.assistant_answer_hint)
                } else {
                    composerPlaceholder
                }
                Composer(
                    value = draft,
                    placeholder = placeholder,
                    notice = notice,
                    sending = sending || assistant.thinking,
                    thinking = assistant.thinking,
                    onStop = if (assistant.thinking) ({ scope.launch(Dispatchers.IO) { runCatching { api.stop() } } }) else null,
                    above = assistant.card?.let { card ->
                        {
                            AssistantAbove(
                                card = card,
                                onUndo = undoCard,
                                onPick = askModel,
                                onSignIn = { assistant.dismiss(); openSync() },
                            )
                        }
                    },
                    chip = if (assistant.model.official && !onTaskPage) ({ ModelChip { assistant.sheet = true } }) else null,
                    onValueChange = { draft = it },
                    onSend = if (route.startsWith("task:")) sendReply else submit,
                    onUndo = noticeUndo,
                    highlights = quickHighlights(activeQuick, draft.length),
                    preview = quickPreview(
                        resources, words, activeQuick,
                        if (route.startsWith("project:")) {
                            projectPage?.project?.name.orEmpty()
                        } else {
                            resources.getString(R.string.title_inbox)
                        },
                    ),
                    onIgnorePreview = { ignoredQuick = draft },
                    focusRequest = composeFocus,
                )
            }
        }
        if (selection != null) {
            BulkBar(
                enabled = selection.orEmpty().isNotEmpty(),
                onDone = { runBulk(BulkAction.Done) },
                onDoToday = { runBulk(BulkAction.DoToday) },
                onMove = {
                    scope.launch {
                        bulkProjects = withContext(Dispatchers.IO) { runCatching { api.projects() }.getOrDefault(emptyList()) }
                        bulkSheet = "move"
                    }
                },
                onPriority = { bulkSheet = "priority" },
                onDelete = { runBulk(BulkAction.Delete) },
            )
        } else if (route == "home" || route == "summary" || route.startsWith("project:") || completedScope != null || isSettingsRoute(route)) {
            BottomNav(
                tab = if (route.startsWith("project:") || (completedScope?.projectId ?: 0L) > 0L || isSettingsRoute(route)) Tab.PROJECTS else tab,
                count = mine?.fromAgents?.size ?: 0,
                onSelect = {
                    tabName = it.name
                    stack = listOf("home")
                },
            )
        }
    }
    }

    if (assistant.sheet) ModelSheet(assistant.model) { assistant.sheet = false }
    if (bulkSheet == "move") {
        BulkMoveSheet(
            projects = bulkProjects,
            onPick = { id, name -> bulkSheet = null; runBulk(BulkAction.MoveTo(id, name)) },
            onDismiss = { bulkSheet = null },
        )
    }
    if (bulkSheet == "priority") {
        BulkPrioritySheet(
            words = words,
            onPick = { priority -> bulkSheet = null; runBulk(BulkAction.SetPriority(priority)) },
            onDismiss = { bulkSheet = null },
        )
    }

    menuData?.let { data ->
        TaskMenuSheet(
            task = data.task,
            words = words,
            agent = data.agent,
            categories = data.categories,
            projects = data.projects,
            presets = data.presets,
            milestones = data.milestones,
            // A board has no checkboxes, so its cards offer no Select (v43: select is the list's).
            showSelect = route == "home" || (
                route.startsWith("project:") &&
                    ProjectViewPrefs.get(context, route.substringAfter(':').toLongOrNull() ?: 0L) == ProjectView.List
                ),
            onSelect = { selection = setOf(data.task.id) },
            initialPage = menuPage,
            onDismiss = { menuData = null },
            onDue = { day ->
                val patch = if (day == null) {
                    TaskPatch(clearDueAt = true)
                } else {
                    TaskPatch(dueAt = dueStamp(day, data.task.dueAt, ZoneId.systemDefault()))
                }
                perform { api.updateTask(data.task.id, patch) }
            },
            onMilestone = { id -> perform { api.updateTask(data.task.id, TaskPatch(milestoneId = id)) } },
            onNewMilestone = { name ->
                val projectId = data.task.projectId
                if (projectId != null) {
                    perform {
                        val made = api.createMilestone(projectId, name)
                        api.updateTask(data.task.id, TaskPatch(milestoneId = made.id))
                    }
                }
            },
            onDoToday = {
                val id = data.task.id
                scope.launch {
                    val before = try {
                        withContext(Dispatchers.IO) { api.doToday(id) }
                    } catch (e: CancellationException) {
                        throw e
                    } catch (e: Exception) {
                        Log.w("adhd-todo", "do today failed", e)
                        null
                    }
                    if (before != null) {
                        toast = UndoToast(resources.getString(R.string.toast_moved_to_today)) {
                            perform { api.restoreDue(id, before) }
                        }
                        tick++
                    }
                }
            },
            onStatus = { status -> perform { api.updateTask(data.task.id, TaskPatch(status = status)) } },
            onPriority = { priority -> perform { api.updateTask(data.task.id, TaskPatch(priority = priority)) } },
            onCategory = { id -> perform { api.updateTask(data.task.id, TaskPatch(category = id.toLong())) } },
            onMove = { projectId ->
                perform { api.updateTask(data.task.id, TaskPatch(projectId = projectId ?: 0L)) }
            },
            onRemind = { at ->
                val patch = if (at == null) {
                    TaskPatch(clearRemindAt = true)
                } else {
                    TaskPatch(
                        remindAt = OffsetDateTime.ofInstant(
                            Instant.ofEpochMilli(at),
                            ZoneId.systemDefault(),
                        ).toString(),
                    )
                }
                perform { api.updateTask(data.task.id, patch) }
            },
            onOpenUrl = openUrl,
            onDelete = { deleteTask(data.task) },
            onNewCategory = { newCategoryFor = data.task.id },
        )
    }

    if (newProject) NewProjectSheet(projectActions, onDismiss = { newProject = false })
    newCategoryFor?.let { taskId ->
        CategorySheet(
            category = null,
            actions = categoryActions,
            onDismiss = { newCategoryFor = null },
            onSaved = { made -> perform { api.updateTask(taskId, TaskPatch(category = made.id)) } },
        )
    }

    replyTo?.let { item ->
        ReplySheet(
            agent = words.agent(mineAgents[item.id]?.agent.orEmpty()),
            onReply = { text ->
                perform { api.reply(item.id, text) }
                replyTo = null
            },
            onDismiss = { replyTo = null },
        )
    }
    if (connect) HowToConnectDialog(onDismiss = { connect = false })

}

/** A snackbar above the composer: a message and what Undo does. */
private data class UndoToast(val message: String, val undo: () -> Unit)

/** What the task menu draws: the fresh task and the three lists its pages need. */
private data class TaskMenuData(
    val task: Task,
    val agent: String,
    val categories: List<Category>,
    val projects: List<Project>,
    val presets: List<RemindPreset>,
    val milestones: List<Milestone> = emptyList(),
)

/** The stack entry of a task: its id and the project it came from (0 for Mine and Agents). */
private fun taskRoute(id: Long, projectId: Long): String =
    if (projectId > 0) "task:$id:$projectId" else "task:$id:0"

/** The launch route of a widget's + and of Note something: Today with the composer focused. */
const val ROUTE_COMPOSE = "compose"

internal fun normalizeRoute(raw: String): String = when {
    raw == ROUTE_COMPOSE -> raw
    raw == "home" || raw == "sync" || raw == "summary" || isSettingsRoute(raw) -> raw
    raw.startsWith("project:") -> raw
    raw.startsWith("task:") -> {
        val id = raw.substringAfter(':').substringBefore(':').toLongOrNull() ?: return ""
        "task:$id:0"
    }
    raw == "projects" -> "tab:projects"
    else -> ""
}

/** The debug build's fixture for one Sync state, for the reviewer's screenshots. */
private fun fakeSyncModel(name: String, resources: Resources): SyncModel {
    val now = Instant.now()
    val mac = Device("mac", "Cyan 的 MacBook Pro", now.minus(3, ChronoUnit.MINUTES).toString(), false)
    val mini = Device("mini", "公司的 Mac mini", now.minus(1, ChronoUnit.DAYS).toString(), false)
    val phone = Device("phone", "Pixel 8", now.toString(), true)
    val devices = listOf(phone, mac, mini)
    val account = Account("Cyan 云辰", "cyan@example.com")
    val base = SyncStatus(
        state = "idle",
        lastSynced = now.toString(),
        server = DEFAULT_SYNC_SERVER,
        linked = true,
        signedIn = true,
    )
    return when (name) {
        "signed_out" -> SyncModel(
            SyncPhase.SIGNED_OUT,
            base.copy(linked = false, signedIn = false, lastSynced = null),
            Account(),
            emptyList(),
        )
        "signed_out_after_linked" -> SyncModel(
            SyncPhase.SIGNED_OUT,
            base.copy(signedIn = false, lastSynced = null),
            Account(),
            emptyList(),
        )
        "signing_in" -> SyncModel(
            SyncPhase.SIGNING_IN,
            base.copy(linked = false, signedIn = true, signInStage = "checking"),
            Account(),
            emptyList(),
        )
        "signing_in_downloading" -> SyncModel(
            SyncPhase.SIGNING_IN,
            base.copy(linked = false, signedIn = true, signInStage = "downloading"),
            Account(),
            emptyList(),
        )
        "choose" -> SyncModel(
            SyncPhase.CHOOSE,
            base.copy(linked = false, signedIn = true),
            account,
            devices.filterNot { it.isThis },
            choose = SignInState(
                state = "choose", localTasks = 128, localProjects = 6, accountDevices = 2,
            ),
        )
        "wrong_account" -> SyncModel(
            SyncPhase.WRONG_ACCOUNT,
            base.copy(state = "wrong account", lastError = "this phone is linked to another account"),
            account,
            devices,
        )
        "other_account" -> SyncModel(
            SyncPhase.OTHER_ACCOUNT,
            base.copy(signedIn = true),
            account,
            emptyList(),
            otherAccount = SignInState(
                state = "other_account", previousEmail = "old@example.com", accountEmail = "cyan@example.com",
            ),
        )
        "google_link" -> SyncModel(
            SyncPhase.SYNCED, base, account, devices, googleLink = GoogleLinkRow(available = true),
        )
        "google_linked" -> SyncModel(
            SyncPhase.SYNCED, base, account, devices, googleLink = GoogleLinkRow(available = true, linked = true),
        )
        "synced" -> SyncModel(SyncPhase.SYNCED, base, account, devices)
        "uploading" -> SyncModel(
            SyncPhase.UPLOADING,
            base.copy(state = "seeding", seedTotal = 3512, pending = 2272, lastSynced = null),
            account,
            devices,
        )
        "failed" -> SyncModel(
            SyncPhase.FAILED,
            base.copy(
                state = "error",
                lastError = "the server did not answer",
                retryAt = now.plus(2, ChronoUnit.MINUTES).toString(),
                failedSince = now.minus(20, ChronoUnit.MINUTES).toString(),
                pending = 3,
            ),
            account,
            devices,
        )
        "offline" -> SyncModel(
            SyncPhase.OFFLINE,
            base.copy(state = "offline", pending = 3),
            account,
            devices,
        )
        else -> SyncModel(SyncPhase.SIGNED_OUT, base.copy(linked = false, signedIn = false), Account(), emptyList())
    }
}
