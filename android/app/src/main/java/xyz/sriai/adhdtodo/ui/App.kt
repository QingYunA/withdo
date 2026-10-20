package xyz.sriai.adhdtodo.ui

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.imePadding
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
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.AgentInfo
import xyz.sriai.adhdtodo.data.AgentsPage
import xyz.sriai.adhdtodo.data.CoreApi
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.core.Core_
import xyz.sriai.adhdtodo.ui.theme.LocalTokens

/**
 * The app shell: header, the visible page, the composer pinned above the bottom bar. The Activity
 * owns the core and its listener; every page is (re)read on a change, off the UI thread, and a
 * failed read leaves the last good page on screen.
 */
@Composable
fun App(core: Core_, revision: Int, syncStatus: String) {
    val context = LocalContext.current
    val resources = context.resources
    val tokens = LocalTokens.current
    val words = remember(resources.configuration) { Words(resources) }
    val api = remember(core) { CoreApi(core) }
    val scope = rememberCoroutineScope()

    var tabName by rememberSaveable { mutableStateOf(Tab.MINE.name) }
    val tab = Tab.valueOf(tabName)
    var tick by remember { mutableIntStateOf(0) }
    var mine by remember { mutableStateOf<MinePage?>(null) }
    var mineAgents by remember { mutableStateOf<Map<Long, AgentInfo>>(emptyMap()) }
    var agents by remember { mutableStateOf<AgentsPage?>(null) }
    var agentsInfo by remember { mutableStateOf<Map<Long, AgentInfo>>(emptyMap()) }

    var draft by rememberSaveable { mutableStateOf("") }
    var sending by remember { mutableStateOf(false) }
    var notice by remember { mutableStateOf<String?>(null) }
    var noticeId by remember { mutableIntStateOf(0) }
    var replyTo by remember { mutableStateOf<Item?>(null) }
    var connect by remember { mutableStateOf(false) }
    var comingSoon by remember { mutableStateOf(false) }

    LaunchedEffect(noticeId) {
        if (notice != null) {
            delay(1500)
            notice = null
        }
    }

    LaunchedEffect(tab, revision, tick) {
        when (tab) {
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
            Tab.CALENDAR, Tab.PROJECTS -> Unit
        }
    }

    fun showNotice(text: String) {
        notice = text
        noticeId++
    }

    fun perform(action: suspend () -> Unit) {
        scope.launch {
            try {
                withContext(Dispatchers.IO) { action() }
                tick++
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                // A short write failure leaves the page as it was; the next change refreshes it.
            }
        }
    }

    val submit: () -> Unit = {
        val title = draft.trim()
        if (title.isNotEmpty() && !sending) {
            sending = true
            scope.launch {
                val ok = try {
                    withContext(Dispatchers.IO) { api.createTask(title) }
                    true
                } catch (e: CancellationException) {
                    throw e
                } catch (_: Exception) {
                    false
                }
                sending = false
                if (ok) {
                    draft = ""
                    showNotice(resources.getString(R.string.added_to_inbox))
                    tick++
                } else {
                    showNotice(resources.getString(R.string.add_failed))
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

    val syncBlocked = remember(syncStatus) {
        Regex("\"state\"\\s*:\\s*\"(error|blocked)\"").containsMatchIn(syncStatus)
    }

    val title = when (tab) {
        Tab.MINE -> stringResource(R.string.title_mine)
        Tab.AGENTS -> stringResource(R.string.title_agents)
        Tab.CALENDAR -> stringResource(R.string.title_calendar)
        Tab.PROJECTS -> stringResource(R.string.title_projects)
    }
    val subtitle = when (tab) {
        Tab.MINE -> words.headerDate()
        Tab.AGENTS -> {
            val working = agents?.working?.size ?: 0
            if (working > 0) resources.getString(R.string.agents_working, working) else ""
        }
        else -> ""
    }
    val placeholder = when (tab) {
        Tab.MINE -> stringResource(R.string.composer_placeholder_mine)
        Tab.AGENTS -> stringResource(R.string.composer_placeholder_agents)
        else -> stringResource(R.string.composer_placeholder_inbox)
    }

    Column(Modifier.fillMaxSize().background(tokens.bg)) {
        Column(Modifier.weight(1f).statusBarsPadding().imePadding()) {
            PageHeader(title = title, subtitle = subtitle, syncBlocked = syncBlocked)
            Box(Modifier.weight(1f)) {
                when (tab) {
                    Tab.MINE -> MineScreen(
                        page = mine,
                        agents = mineAgents,
                        words = words,
                        onAccept = { id -> perform { api.accept(id) } },
                        onComplete = { id -> perform { api.complete(id) } },
                        onAnswer = { replyTo = it },
                        onOpenUrl = openUrl,
                        onOpenTask = { comingSoon = true },
                        onHowToConnect = { connect = true },
                        onComingSoon = { comingSoon = true },
                    )
                    Tab.AGENTS -> AgentsScreen(
                        page = agents,
                        agents = agentsInfo,
                        words = words,
                        onOpenTask = { comingSoon = true },
                        onHowToConnect = { connect = true },
                    )
                    Tab.CALENDAR -> Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
                        PlaceholderScreen(
                            icon = "calendar",
                            title = stringResource(R.string.calendar_placeholder_title),
                            body = stringResource(R.string.calendar_placeholder_body),
                        )
                    }
                    Tab.PROJECTS -> Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
                        PlaceholderScreen(
                            icon = "folder",
                            title = stringResource(R.string.projects_placeholder_title),
                            body = stringResource(R.string.projects_placeholder_body),
                        )
                    }
                }
            }
            Composer(
                value = draft,
                placeholder = placeholder,
                notice = notice,
                sending = sending,
                onValueChange = { draft = it },
                onSend = submit,
            )
        }
        BottomNav(
            tab = tab,
            // The red badge counts what agents handed over (ui-spec: Sidebar). The phone has no
            // seen-state yet, so every handed-over row counts as unseen.
            count = mine?.fromAgents?.size ?: 0,
            onSelect = { tabName = it.name },
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
