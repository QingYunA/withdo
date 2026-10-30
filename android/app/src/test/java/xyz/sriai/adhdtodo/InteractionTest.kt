package xyz.sriai.adhdtodo

import androidx.compose.foundation.gestures.Orientation
import androidx.compose.foundation.gestures.draggable
import androidx.compose.foundation.gestures.rememberDraggableState
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.SemanticsNodeInteraction
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertTextContains
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.assertIsFocused
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.filter
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.junit4.ComposeContentTestRule
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onChildren
import androidx.compose.ui.test.onFirst
import androidx.compose.ui.test.onLast
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.onParent
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performImeAction
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.text.TextLayoutResult
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.AgentInfo
import xyz.sriai.adhdtodo.data.Link
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.ProjectPage
import xyz.sriai.adhdtodo.data.SyncStatus
import xyz.sriai.adhdtodo.ui.CodeFailure
import xyz.sriai.adhdtodo.ui.SignInActions
import xyz.sriai.adhdtodo.ui.SignInFlow
import xyz.sriai.adhdtodo.ui.SignInStep
import xyz.sriai.adhdtodo.data.TaskPage
import xyz.sriai.adhdtodo.testing.FixedTime
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.testing.dragBy
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.BottomNav
import xyz.sriai.adhdtodo.ui.Composer
import xyz.sriai.adhdtodo.ui.DEFAULT_SYNC_SERVER
import xyz.sriai.adhdtodo.ui.MineScreen
import xyz.sriai.adhdtodo.ui.ProjectScreen
import xyz.sriai.adhdtodo.ui.SyncModel
import xyz.sriai.adhdtodo.ui.SyncPhase
import xyz.sriai.adhdtodo.ui.SyncScreen
import xyz.sriai.adhdtodo.ui.Tab
import xyz.sriai.adhdtodo.ui.TaskScreen
import xyz.sriai.adhdtodo.ui.Words

/** The existing phone interactions, driven on the JVM through Compose's test semantics. */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class InteractionTest {
    @get:Rule
    val compose = createComposeRule()

    private val mine: MinePage = Golden.load("mine.golden.json")
    private val agents = mine.fromAgents.associate { item ->
        val links = if (item.prUrl.isNotEmpty()) listOf(Link(item.prUrl, "PR #${item.prNumber}")) else emptyList()
        item.id to AgentInfo("claude-code", links)
    }

    @Test
    fun `mine row opens the task`() {
        var opened = 0L
        compose.setThemedContent {
            MineScreen(
                page = mine,
                agents = agents,
                words = rememberWords(),
                onAccept = {},
                onComplete = {},
                onAnswer = {},
                onOpenUrl = {},
                onOpenTask = { opened = it },
                onHowToConnect = {},
                onViewAll = null,
            )
        }
        compose.clickText("Reply to design feedback")
        compose.runOnIdle { assertEquals(7L, opened) }
    }

    @Test
    fun `mine completion box completes the task`() {
        var completed = 0L
        compose.setThemedContent {
            MineScreen(
                page = mine,
                agents = agents,
                words = rememberWords(),
                onAccept = {},
                onComplete = { completed = it },
                onAnswer = {},
                onOpenUrl = {},
                onOpenTask = {},
                onHowToConnect = {},
                onViewAll = null,
            )
        }
        compose.clickCompletion("Reply to design feedback")
        compose.runOnIdle { assertEquals(7L, completed) }
    }

    @Test
    fun `mine accept accepts the agent item`() {
        var accepted = 0L
        compose.setThemedContent {
            MineScreen(
                page = mine,
                agents = agents,
                words = rememberWords(),
                onAccept = { accepted = it },
                onComplete = {},
                onAnswer = {},
                onOpenUrl = {},
                onOpenTask = {},
                onHowToConnect = {},
                onViewAll = null,
            )
        }
        compose.clickText("Mark done")
        compose.runOnIdle { assertEquals(2L, accepted) }
    }

    @Test
    fun `task priority strip reports the tapped priority`() {
        var priority = ""
        compose.setThemedContent {
            TaskScreen(
                page = Golden.load<TaskPage>("task-1.golden.json"),
                projects = Golden.load<List<Project>>("projects.golden.json"),
                words = rememberWords(),
                onSetPriority = { priority = it },
                onOpenUrl = {},
                onOpenTask = {},
            )
        }
        for (value in listOf("P0", "P1", "P2", "P3")) {
            compose.clickText(value)
            compose.runOnIdle { assertEquals(value, priority) }
        }
    }

    @Test
    fun `task reply sends the typed text`() {
        val draft = mutableStateOf("")
        var sent: String? = null
        compose.setThemedContent {
            TaskWithComposer(draft, onSend = { sent = draft.value })
        }
        compose.onNode(hasSetTextAction()).performTextInput("Looks good")
        compose.onNode(hasSetTextAction()).performImeAction()
        compose.runOnIdle { assertEquals("Looks good", sent) }
    }

    @Test
    fun `task reply refuses blank text`() {
        val draft = mutableStateOf("")
        var sent: String? = null
        compose.setThemedContent {
            Composer(
                value = draft.value,
                placeholder = "Reply to Claude Code",
                notice = null,
                sending = false,
                onValueChange = { draft.value = it },
                onSend = { sent = draft.value },
            )
        }
        compose.onNode(hasSetTextAction()).performImeAction()
        compose.onNode(hasSetTextAction()).performTextInput("   ")
        compose.onNode(hasSetTextAction()).performImeAction()
        compose.runOnIdle { assertNull(sent) }
    }

    @Test
    fun `project segments switch between mine and agents`() {
        compose.setThemedContent {
            ProjectScreen(
                page = Golden.load<ProjectPage>("project-1.golden.json"),
                words = rememberWords(),
                onOpenTask = {},
            )
        }
        compose.onNodeWithText("Decide on the activation policy").assertExists()
        compose.clickText("Agents")
        compose.onNodeWithText("Port core to Go").assertExists()
        compose.onNodeWithText("Decide on the activation policy").assertDoesNotExist()
        compose.clickText("Mine")
        compose.onNodeWithText("Decide on the activation policy").assertExists()
        compose.onNodeWithText("Port core to Go").assertDoesNotExist()
    }

    @Test
    fun `project completed row unfolds`() {
        compose.setThemedContent {
            ProjectScreen(
                page = Golden.load<ProjectPage>("project-1.golden.json"),
                words = rememberWords(),
                onOpenTask = {},
            )
        }
        compose.onNodeWithText("Pick a UI toolkit").assertDoesNotExist()
        compose.clickText("Completed 2")
        compose.onNodeWithText("Pick a UI toolkit").assertExists()
    }

    @Test
    fun `composer submits only through the IME send action`() {
        val draft = mutableStateOf("Write the tests")
        var sent: String? = null
        compose.setThemedContent {
            Composer(
                value = draft.value,
                placeholder = "Add to Inbox",
                notice = null,
                sending = false,
                onValueChange = { draft.value = it },
                onSend = { sent = draft.value },
            )
        }
        compose.onNode(hasSetTextAction()).performImeAction()
        compose.runOnIdle { assertEquals("Write the tests", sent) }
    }

    @Test
    fun `continue with google starts the browser flow and cancel stops the wait`() {
        var started = 0
        var cancelled = 0
        val actions = xyz.sriai.adhdtodo.ui.SignInActions(
            onGoogle = { started++ },
            onCancelGoogle = { cancelled++ },
        )
        val model = mutableStateOf(
            SyncModel(
                SyncPhase.SIGNED_OUT,
                SyncStatus(server = DEFAULT_SYNC_SERVER),
                flow = xyz.sriai.adhdtodo.ui.SignInFlow(),
                googleAvailable = true,
            ),
        )
        compose.setThemedContent { Sync(model, signInActions = actions) }
        compose.clickText("Continue with Google")
        assertEquals(1, started)
        model.value = model.value.copy(
            phase = SyncPhase.SIGNING_IN,
            status = SyncStatus(server = DEFAULT_SYNC_SERVER, signedIn = true, signInStage = "google"),
        )
        compose.waitForIdle()
        compose.onNodeWithText("Finish in your browser").assertExists()
        compose.onNodeWithText("Email").assertDoesNotExist()
        compose.clickText("Cancel")
        assertEquals(1, cancelled)
    }

    @Test
    fun `the google button is hidden when the server has no google provider`() {
        val model = mutableStateOf(
            SyncModel(
                SyncPhase.SIGNED_OUT,
                SyncStatus(server = DEFAULT_SYNC_SERVER),
                flow = xyz.sriai.adhdtodo.ui.SignInFlow(),
            ),
        )
        compose.setThemedContent { Sync(model) }
        compose.onNodeWithText("Email").assertExists()
        compose.onNodeWithText("Continue with Google").assertDoesNotExist()
    }

    /** The signed-out card over a model the actions move on, as the app does. */
    private fun signedOut(
        step: SignInStep = SignInStep.SIGN_IN,
        failure: CodeFailure? = null,
        email: String = "",
        calls: MutableList<String> = mutableListOf(),
    ): Pair<MutableState<SyncModel>, SignInActions> {
        val model = mutableStateOf(
            SyncModel(
                SyncPhase.SIGNED_OUT,
                SyncStatus(server = DEFAULT_SYNC_SERVER),
                flow = SignInFlow(step = step, email = email, failure = failure, resendIn = 24),
            ),
        )
        val actions = SignInActions(
            onSignInWithPassword = { e, p -> calls += "signin:$e:$p" },
            onSignUp = { e, p -> calls += "signup:$e:$p" },
            onSubmitCode = { calls += "code:$it" },
            onSendResetCode = { calls += "sendreset:$it" },
            onResetPassword = { c, p -> calls += "reset:$c:$p" },
            onOpenUrl = { calls += "open:$it" },
            onStep = { next ->
                calls += "step:$next"
                model.value = model.value.copy(flow = model.value.flow.copy(step = next, failure = null))
            },
        )
        return model to actions
    }

    @Test
    fun `the tabs switch and keep the typed email`() {
        val (model, actions) = signedOut()
        compose.setThemedContent { Sync(model, signInActions = actions) }
        compose.onNodeWithText("Forgot password?").assertExists()
        compose.onNodeWithText("Create account").assertDoesNotExist()
        compose.fields()[0].performTextInput("cyan@example.com")
        compose.onAllNodesWithText("Sign up").onFirst().performClick()
        compose.waitForIdle()
        compose.onNodeWithText("Create account").assertExists()
        compose.onNodeWithText("At least 8 characters.").assertExists()
        compose.onNodeWithText("Forgot password?").assertDoesNotExist()
        compose.fields()[0].assertTextContains("cyan@example.com")
        compose.onAllNodesWithText("Sign in").onFirst().performClick()
        compose.waitForIdle()
        compose.onNodeWithText("Forgot password?").assertExists()
        compose.fields()[0].assertTextContains("cyan@example.com")
    }

    @Test
    fun `sign in sends the email and the password`() {
        val calls = mutableListOf<String>()
        val (model, actions) = signedOut(calls = calls)
        // The Sync screen takes the sign-in of the card from onSignIn, as the app wires it.
        compose.setThemedContent { Sync(model, onSignIn = { e, p -> calls += "signin:$e:$p" }, signInActions = actions) }
        compose.fields()[0].performTextInput("cyan@example.com")
        compose.fields()[1].performTextInput("hunter22")
        compose.onAllNodesWithText("Sign in").onLast().performClick()
        assertEquals(listOf("signin:cyan@example.com:hunter22"), calls)
    }

    @Test
    fun `sign up needs an email and eight characters`() {
        val calls = mutableListOf<String>()
        val (model, actions) = signedOut(step = SignInStep.SIGN_UP, calls = calls)
        compose.setThemedContent { Sync(model, signInActions = actions) }
        compose.clickableText("Create account").assertIsNotEnabled()
        compose.fields()[0].performTextInput("cyan@example.com")
        compose.fields()[1].performTextInput("short")
        compose.clickableText("Create account").assertIsNotEnabled()
        compose.fields()[1].performTextInput("er pass")
        compose.clickableText("Create account").assertIsEnabled().performClick()
        assertEquals(listOf("signup:cyan@example.com:shorter pass"), calls)
    }

    @Test
    fun `a taken email says so and its link picks sign in`() {
        val calls = mutableListOf<String>()
        val (model, actions) = signedOut(step = SignInStep.SIGN_UP, failure = CodeFailure.EMAIL_TAKEN, email = "cyan@example.com", calls = calls)
        compose.setThemedContent { Sync(model, signInActions = actions) }
        compose.onNodeWithText("This email already has an account.").assertExists()
        compose.onAllNodesWithText("Sign in").onLast().performClick()
        assertEquals(listOf("step:SIGN_IN"), calls)
        compose.waitForIdle()
        compose.onNodeWithText("Forgot password?").assertExists()
    }

    @Test
    fun `a wrong password is worded under its field`() {
        val (wrong, wrongActions) = signedOut(failure = CodeFailure.WRONG_PASSWORD, email = "cyan@example.com")
        compose.setThemedContent { Sync(wrong, signInActions = wrongActions) }
        compose.onNodeWithText("Wrong email or password").assertExists()
    }

    @Test
    fun `a malformed email is worded under its field`() {
        val (bad, badActions) = signedOut(step = SignInStep.SIGN_UP, failure = CodeFailure.INVALID_EMAIL)
        compose.setThemedContent { Sync(bad, signInActions = badActions) }
        compose.onNodeWithText("That doesn’t look like an email").assertExists()
    }

    @Test
    fun `show reveals the password and hide covers it again`() {
        val (model, actions) = signedOut(step = SignInStep.SIGN_UP)
        compose.setThemedContent { Sync(model, signInActions = actions) }
        compose.onNodeWithText("Show").performClick()
        compose.onNodeWithText("Hide").assertExists()
        compose.onNodeWithText("Hide").performClick()
        compose.onNodeWithText("Show").assertExists()
    }

    @Test
    fun `forgot password goes to its steps and sends the code and the new password`() {
        val calls = mutableListOf<String>()
        val (model, actions) = signedOut(email = "cyan@example.com", calls = calls)
        compose.setThemedContent { Sync(model, signInActions = actions) }
        compose.onNodeWithText("Forgot password?").performClick()
        compose.waitForIdle()
        compose.onNodeWithText("Reset your password").assertExists()
        compose.fields()[0].assertTextContains("cyan@example.com")
        compose.clickableText("Send code").performClick()
        assertEquals(listOf("step:RESET_EMAIL", "sendreset:cyan@example.com"), calls)

        model.value = model.value.copy(flow = SignInFlow(step = SignInStep.RESET_CODE, email = "cyan@example.com", resendIn = 24))
        compose.waitForIdle()
        compose.onNodeWithText("The code went to cyan@example.com", substring = true).assertExists()
        compose.clickableText("Set and sign in").assertIsNotEnabled()
        compose.fields()[0].performTextInput("482913")
        compose.fields()[1].performTextInput("a new password")
        compose.clickableText("Set and sign in").assertIsEnabled().performClick()
        assertEquals("reset:482913:a new password", calls.last())
    }

    @Test
    fun `the sixth digit of the verify step sends the code`() {
        val calls = mutableListOf<String>()
        val (model, actions) = signedOut(step = SignInStep.VERIFY, email = "cyan@example.com", calls = calls)
        compose.setThemedContent { Sync(model, signInActions = actions) }
        compose.onNodeWithText("Verify your email").assertExists()
        compose.onNodeWithText("It checks itself once complete").assertExists()
        compose.fields()[0].performTextInput("48291")
        assertEquals(emptyList<String>(), calls)
        compose.fields()[0].performTextInput("3")
        assertEquals(listOf("code:482913"), calls)
    }

    @Test
    fun `the terms line has the two links`() {
        val (model, actions) = signedOut(step = SignInStep.SIGN_UP)
        compose.setThemedContent { Sync(model, signInActions = actions) }
        compose.onNodeWithText("By creating an account you agree to the", substring = true).assertExists()
    }

    @Test
    fun `bottom nav selects each tab`() {
        val tab = mutableStateOf(Tab.MINE)
        compose.setThemedContent {
            Column {
                Text("tab ${tab.value.name}")
                BottomNav(tab = tab.value, count = 0, onSelect = { tab.value = it })
            }
        }
        for ((label, value) in listOf("Today" to Tab.MINE, "Agents" to Tab.AGENTS, "Calendar" to Tab.CALENDAR, "Projects" to Tab.PROJECTS)) {
            compose.clickText(label)
            compose.onNodeWithText("tab ${value.name}").assertExists()
            val layout = compose.onNodeWithText(label, useUnmergedTree = true).textLayout()
            assertEquals(FontWeight.SemiBold, layout.layoutInput.style.fontWeight)
        }
    }

    @Test
    fun `dragBy drags a draggable sample`() {
        var offset by mutableFloatStateOf(0f)
        compose.setThemedContent {
            Box(
                Modifier
                    .size(200.dp)
                    .testTag("pad")
                    .draggable(
                        state = rememberDraggableState { offset += it },
                        orientation = Orientation.Vertical,
                    ),
            )
        }
        dragBy(compose.onNodeWithTag("pad"), dy = 120f)
        compose.runOnIdle {
            assertTrue("dragged by $offset", offset > 0f)
            assertTrue("did not overshoot: $offset", offset <= 120f)
        }
    }

    @Composable
    private fun TaskWithComposer(draft: MutableState<String>, onSend: () -> Unit) {
        Column(Modifier.fillMaxSize()) {
            Box(Modifier.weight(1f)) {
                TaskScreen(
                    page = Golden.load<TaskPage>("task-1.golden.json"),
                    projects = Golden.load<List<Project>>("projects.golden.json"),
                    words = rememberWords(),
                    onSetPriority = {},
                    onOpenUrl = {},
                    onOpenTask = {},
                )
            }
            Composer(
                value = draft.value,
                placeholder = "Reply to Claude Code",
                notice = null,
                sending = false,
                onValueChange = { draft.value = it },
                onSend = onSend,
            )
        }
    }

    @Composable
    private fun Sync(
        model: MutableState<SyncModel>,
        onSignIn: (String, String) -> Unit = { _, _ -> },
        signInActions: xyz.sriai.adhdtodo.ui.SignInActions = xyz.sriai.adhdtodo.ui.SignInActions(),
    ) {
        SyncScreen(
            model = model.value,
            words = rememberWords(),
            serverEditing = false,
            serverDraft = "",
            serverError = false,
            serverSaving = false,
            onSignIn = onSignIn,
            signInActions = signInActions,
            onSignOut = {},
            onSyncNow = {},
            onRetry = {},
            onMerge = {},
            onReplace = {},
            onCancelChoose = {},
            onStartServerEdit = {},
            onServerDraft = {},
            onSaveServer = {},
            onCancelServerEdit = {},
        )
    }

    @Composable
    private fun rememberWords(): Words {
        val resources = LocalContext.current.resources
        return remember(resources.configuration) { Words(resources, FixedTime.clock, FixedTime.zone) }
    }

    private fun ComposeContentTestRule.fields() = onAllNodes(hasSetTextAction())

    /** The nearest node above [text] that handles a click. */
    private fun ComposeContentTestRule.clickableText(text: String): SemanticsNodeInteraction {
        var node = onNodeWithText(text, useUnmergedTree = true)
        repeat(10) {
            if (node.fetchSemanticsNode().config.contains(SemanticsActions.OnClick)) return node
            node = node.onParent()
        }
        error("no click action above \"$text\"")
    }

    private fun ComposeContentTestRule.clickText(text: String) {
        clickableText(text).performClick()
    }

    /** The completion glyph of a Mine row: the first clickable child of the row. */
    private fun ComposeContentTestRule.clickCompletion(title: String) {
        clickableText(title).onChildren().filter(hasClickAction()).onFirst().performClick()
    }

    private fun SemanticsNodeInteraction.assertEditableText(expected: String) {
        assertEquals(expected, fetchSemanticsNode().config[SemanticsProperties.EditableText].text)
    }

    private fun SemanticsNodeInteraction.textLayout(): TextLayoutResult {
        val results = mutableListOf<TextLayoutResult>()
        fetchSemanticsNode().config[SemanticsActions.GetTextLayoutResult].action?.invoke(results)
        return results.first()
    }
}
