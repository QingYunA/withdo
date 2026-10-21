package xyz.sriai.adhdtodo.ui

import androidx.activity.compose.BackHandler
import android.content.Context
import android.content.res.Resources
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.util.Log
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
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.listSaver
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import xyz.sriai.adhdtodo.BuildConfig
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Account
import xyz.sriai.adhdtodo.data.AgentInfo
import xyz.sriai.adhdtodo.data.AgentsPage
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.CoreApi
import xyz.sriai.adhdtodo.data.Device
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.MinePage
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
import xyz.sriai.adhdtodo.core.Core_
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import java.time.Instant
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

    var mine by remember { mutableStateOf<MinePage?>(null) }
    var mineAgents by remember { mutableStateOf<Map<Long, AgentInfo>>(emptyMap()) }
    var agents by remember { mutableStateOf<AgentsPage?>(null) }
    var agentsInfo by remember { mutableStateOf<Map<Long, AgentInfo>>(emptyMap()) }
    var projectEntries by remember { mutableStateOf<List<ProjectEntry>?>(null) }
    var projectPage by remember { mutableStateOf<ProjectPage?>(null) }
    var taskPage by remember { mutableStateOf<TaskPage?>(null) }
    var projects by remember { mutableStateOf<List<Project>>(emptyList()) }
    var categories by remember { mutableStateOf<List<Category>>(emptyList()) }
    var account by remember { mutableStateOf<Account?>(null) }
    var devices by remember { mutableStateOf<List<Device>>(emptyList()) }
    var liveStatus by remember { mutableStateOf<SyncStatus?>(null) }

    var draft by rememberSaveable { mutableStateOf("") }
    var sending by remember { mutableStateOf(false) }
    var notice by remember { mutableStateOf<String?>(null) }
    var noticeId by remember { mutableIntStateOf(0) }
    var replyTo by remember { mutableStateOf<Item?>(null) }
    var connect by remember { mutableStateOf(false) }
    var comingSoon by remember { mutableStateOf(false) }
    var menuData by remember { mutableStateOf<TaskMenuData?>(null) }
    var menuPage by remember { mutableStateOf(TaskMenuPage.Main) }
    var deleted by remember { mutableStateOf<DeletedTask?>(null) }

    var signingIn by remember { mutableStateOf(false) }
    var choosing by remember { mutableStateOf(false) }
    var serverEditing by remember { mutableStateOf(false) }
    var serverDraft by remember { mutableStateOf("") }
    var serverError by remember { mutableStateOf(false) }
    var serverSaving by remember { mutableStateOf(false) }
    var syncMessage by remember { mutableStateOf<String?>(null) }
    var signInError by remember { mutableStateOf<String?>(null) }
    var signInFailure by remember { mutableStateOf<SignInFailure?>(null) }
    var choose by remember { mutableStateOf(SignInState()) }

    val route = stack.last()
    val tab = runCatching { Tab.valueOf(tabName) }.getOrDefault(Tab.MINE)
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

    fun showNotice(text: String) {
        notice = text
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
                deleted = DeletedTask(task.title, snapshot)
                if (onTaskPage) pop()
                tick++
            }
        }
    }

    BackHandler(enabled = stack.size > 1) { pop() }

    LaunchedEffect(noticeId) {
        if (notice != null) {
            delay(1500)
            notice = null
        }
    }

    LaunchedEffect(deleted) {
        if (deleted == null) return@LaunchedEffect
        delay(6000)
        deleted = null
    }

    LaunchedEffect(initialRoute) {
        if (initialRoute.isNullOrBlank()) return@LaunchedEffect
        val next = normalizeRoute(initialRoute)
        if (next.startsWith("tab:")) {
            // route=projects opens the Projects tab, not Mine with the projects route on top.
            tabName = runCatching { Tab.valueOf(next.removePrefix("tab:").uppercase()) }
                .getOrDefault(Tab.MINE).name
            stack = listOf("home")
        } else if (next.isNotBlank()) {
            stack = if (next == "home") listOf("home") else listOf("home", next)
        }
        onInitialRouteConsumed()
    }

    LaunchedEffect(route, revision, tick, tab) {
        when {
            route == "home" -> when (tab) {
                Tab.MINE -> {
                    val loaded = try {
                        withContext(Dispatchers.IO) {
                            val page = api.mine()
                            page to api.agentInfo(page)
                        }
                    } catch (e: CancellationException) {
                        throw e
                    } catch (_: Exception) {
                        null
                    }
                    if (loaded != null) {
                        mine = loaded.first
                        mineAgents = loaded.second
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
                        withContext(Dispatchers.IO) { api.projectEntries() }
                    } catch (e: CancellationException) {
                        throw e
                    } catch (_: Exception) {
                        null
                    }
                    if (loaded != null) projectEntries = loaded
                }
                Tab.CALENDAR -> Unit
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

    fun startSync() {
        scope.launch {
            runCatching { withContext(Dispatchers.IO) { api.startSync() } }
        }
    }

    fun undoDelete(item: DeletedTask) {
        deleted = null
        perform { api.restoreTask(item.snapshot) }
    }

    val submit: () -> Unit = {
        val title = draft.trim()
        val projectId = if (route.startsWith("project:")) route.substringAfter(':').toLongOrNull() else null
        if (title.isNotEmpty() && !sending) {
            sending = true
            scope.launch {
                val ok = try {
                    withContext(Dispatchers.IO) { api.createTask(title, projectId) }
                    true
                } catch (e: CancellationException) {
                    throw e
                } catch (_: Exception) {
                    false
                }
                sending = false
                if (ok) {
                    draft = ""
                    showNotice(
                        if (projectId != null) {
                            resources.getString(R.string.added_to_project, projectPage?.project?.name.orEmpty())
                        } else {
                            resources.getString(R.string.added_to_inbox)
                        },
                    )
                    tick++
                } else {
                    showNotice(resources.getString(R.string.add_failed))
                }
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

    val title = when {
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
        tab == Tab.MINE -> words.headerDate()
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
        tab == Tab.MINE -> stringResource(R.string.composer_placeholder_mine)
        tab == Tab.AGENTS -> stringResource(R.string.composer_placeholder_agents)
        else -> stringResource(R.string.composer_placeholder_inbox)
    }

    Column(Modifier.fillMaxSize().background(tokens.bg)) {
        Column(Modifier.weight(1f).statusBarsPadding().imePadding()) {
            if (route == "home") {
                PageHeader(title = title, subtitle = subtitle, syncBlocked = blocked, onSync = { openSync() })
            } else if (route == "sync") {
                BackBar(title = stringResource(R.string.sync_title), onBack = { pop() })
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

            Box(Modifier.weight(1f)) {
                when {
                    route == "sync" -> {
                        val model = fake ?: SyncModel(
                            phase = syncPhaseOf(shownStatus, signingIn, choosing),
                            status = shownStatus,
                            account = account ?: Account(),
                            devices = devices,
                            choose = choose,
                            signInError = signInError,
                            signInFailure = signInFailure,
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
                            SyncScreen(
                                model = model,
                                words = words,
                                serverEditing = serverEditing,
                                serverDraft = serverDraft,
                                serverError = serverError,
                                serverSaving = serverSaving,
                                onSignIn = { login, password ->
                                    if (!signingIn) {
                                        signingIn = true
                                        choosing = false
                                        signInError = null
                                        signInFailure = null
                                        syncMessage = null
                                        scope.launch {
                                            var failure: Exception? = null
                                            val result = try {
                                                withContext(Dispatchers.IO) {
                                                    api.signIn(login, password, Build.MODEL)
                                                }
                                            } catch (e: CancellationException) {
                                                throw e
                                            } catch (e: Exception) {
                                                failure = e
                                                null
                                            }
                                            signingIn = false
                                            if (failure != null) {
                                                Log.w("adhd-todo", "sign-in failed", failure)
                                                val kind = signInFailureOf(failure.message)
                                                signInFailure = kind
                                                signInError = if (kind != null) {
                                                    resources.getString(signInFailureText(kind))
                                                } else {
                                                    signInFailureSentence(failure)
                                                        ?: resources.getString(R.string.sync_sign_in_failed)
                                                }
                                            }
                                            when (result?.state) {
                                                "choose" -> {
                                                    choose = checkNotNull(result)
                                                    choosing = true
                                                }
                                                "linked" -> startSync()
                                                null -> if (failure == null) {
                                                    syncMessage = resources.getString(R.string.sync_sign_in_failed)
                                                }
                                                else -> Unit
                                            }
                                            tick++
                                        }
                                    }
                                },
                                onSignOut = {
                                    choosing = false
                                    signingIn = false
                                    signInError = null
                                    signInFailure = null
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
                                    perform { api.signOut() }
                                },
                                onStartServerEdit = {
                                    serverEditing = true
                                    serverDraft = shownStatus.server.ifBlank { DEFAULT_SYNC_SERVER }
                                    serverError = false
                                },
                                onServerDraft = { serverDraft = it; serverError = false },
                                onSaveServer = {
                                    if (!serverSaving) {
                                        serverSaving = true
                                        serverError = false
                                        scope.launch {
                                            val ok = try {
                                                withContext(Dispatchers.IO) { api.syncSetServer(serverDraft.trim()) }
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
                    route.startsWith("project:") -> ProjectScreen(
                        page = projectPage,
                        words = words,
                        onOpenTask = { id -> push(taskRoute(id, route.substringAfter(':').toLongOrNull() ?: 0)) },
                        onLongPress = { id -> openTaskMenu(id) },
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
                    )
                    else -> when (tab) {
                        Tab.MINE -> MineScreen(
                            page = mine,
                            agents = mineAgents,
                            words = words,
                            onAccept = { id -> perform { api.accept(id) } },
                            onComplete = { id -> perform { api.complete(id) } },
                            onAnswer = { replyTo = it },
                            onOpenUrl = openUrl,
                            onOpenTask = { id -> push(taskRoute(id, 0)) },
                            onHowToConnect = { connect = true },
                            onComingSoon = { comingSoon = true },
                            onLongPress = { id -> openTaskMenu(id) },
                        )
                        Tab.AGENTS -> AgentsScreen(
                            page = agents,
                            agents = agentsInfo,
                            words = words,
                            onOpenTask = { id -> push(taskRoute(id, 0)) },
                            onHowToConnect = { connect = true },
                            onLongPress = { id -> openTaskMenu(id) },
                        )
                        Tab.CALENDAR -> Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
                            PlaceholderScreen(
                                icon = "calendar",
                                title = stringResource(R.string.calendar_placeholder_title),
                                body = stringResource(R.string.calendar_placeholder_body),
                            )
                        }
                        Tab.PROJECTS -> ProjectsScreen(
                            entries = projectEntries,
                            onOpenProject = { id -> push("project:$id") },
                            onHowToConnect = { connect = true },
                        )
                    }
                }
            }

            DeletedSnackbar(deleted = deleted, onUndo = { undoDelete(it) })

            if (composerVisible) {
                val placeholder = composerPlaceholder
                Composer(
                    value = draft,
                    placeholder = placeholder,
                    notice = notice,
                    sending = sending,
                    onValueChange = { draft = it },
                    onSend = if (route.startsWith("task:")) sendReply else submit,
                )
            }
        }
        if (route == "home" || route.startsWith("project:")) {
            BottomNav(
                tab = if (route.startsWith("project:")) Tab.PROJECTS else tab,
                count = mine?.fromAgents?.size ?: 0,
                onSelect = {
                    tabName = it.name
                    stack = listOf("home")
                },
            )
        }
    }

    menuData?.let { data ->
        TaskMenuSheet(
            task = data.task,
            words = words,
            agent = data.agent,
            categories = data.categories,
            projects = data.projects,
            presets = data.presets,
            initialPage = menuPage,
            onDismiss = { menuData = null },
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
    if (comingSoon) TaskSoonDialog(onDismiss = { comingSoon = false })

}

/** What the task menu draws: the fresh task and the three lists its pages need. */
private data class TaskMenuData(
    val task: Task,
    val agent: String,
    val categories: List<Category>,
    val projects: List<Project>,
    val presets: List<RemindPreset>,
)

/** The stack entry of a task: its id and the project it came from (0 for Mine and Agents). */
private fun taskRoute(id: Long, projectId: Long): String =
    if (projectId > 0) "task:$id:$projectId" else "task:$id:0"

internal fun normalizeRoute(raw: String): String = when {
    raw == "home" || raw == "sync" -> raw
    raw.startsWith("project:") -> raw
    raw.startsWith("task:") -> {
        val id = raw.substringAfter(':').substringBefore(':').toLongOrNull() ?: return ""
        "task:$id:0"
    }
    raw == "projects" -> "tab:projects"
    else -> ""
}

/** The Go error's sentence for a failure the app has no localized sentence for. */
internal fun signInFailureSentence(error: Throwable?): String? =
    error?.message?.trim()?.takeIf { it.isNotEmpty() }

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
        "signed_out_error" -> SyncModel(
            SyncPhase.SIGNED_OUT,
            base.copy(linked = false, signedIn = false, lastSynced = null),
            Account(),
            emptyList(),
            signInError = resources.getString(R.string.sync_sign_in_invalid),
            signInFailure = SignInFailure.CREDENTIALS,
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
        "sign_in_failed" -> SyncModel(
            SyncPhase.SIGNED_OUT,
            base.copy(linked = false, signedIn = false, lastSynced = null),
            Account(),
            emptyList(),
            signInError = "could not reach the sign-in server",
        )
        else -> SyncModel(SyncPhase.SIGNED_OUT, base.copy(linked = false, signedIn = false), Account(), emptyList())
    }
}
