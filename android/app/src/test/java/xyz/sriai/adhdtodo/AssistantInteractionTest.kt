package xyz.sriai.adhdtodo

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.size
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onAllNodesWithTag
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.unit.dp
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.AssistantQuestion
import xyz.sriai.adhdtodo.data.ModelPhase
import xyz.sriai.adhdtodo.data.ModelState
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.AnswerCard
import xyz.sriai.adhdtodo.ui.AnswerRow
import xyz.sriai.adhdtodo.ui.AssistantAbove
import xyz.sriai.adhdtodo.ui.CardUndo
import xyz.sriai.adhdtodo.ui.Composer
import xyz.sriai.adhdtodo.ui.ModelChip
import xyz.sriai.adhdtodo.ui.ModelSheet
import xyz.sriai.adhdtodo.ui.QuietNote
import xyz.sriai.adhdtodo.ui.dismissOnTouch

/** The assistant's touches (v44): Undo, an option, Stop, the chip and a touch outside the card. */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class AssistantInteractionTest {
    @get:Rule
    val compose = createComposeRule()

    private val receipts = AnswerCard(
        summary = "Added 1",
        rows = listOf(AnswerRow(2, "Call the landlord", listOf("Tomorrow 9:00 AM"), "todo")),
        undo = CardUndo.Receipts(listOf(1)),
    )

    @Test
    fun `undo on the card is one tap`() {
        var undone = 0
        compose.setThemedContent { AssistantAbove(receipts, onUndo = { undone++ }, onPick = {}, onSignIn = {}) }
        compose.onNodeWithText("Added 1").assertIsDisplayed()
        compose.onNodeWithText("Call the landlord").assertIsDisplayed()
        compose.onNodeWithTag("assistant-undo").performClick()
        assertEquals(1, undone)
    }

    @Test
    fun `a plain answer has no undo`() {
        compose.setThemedContent { AssistantAbove(AnswerCard(text = "Five things."), onUndo = {}, onPick = {}, onSignIn = {}) }
        compose.onNodeWithText("Five things.").assertIsDisplayed()
        compose.onNodeWithTag("assistant-undo").assertDoesNotExist()
    }

    @Test
    fun `a tap on an option sends that option`() {
        val picked = mutableListOf<String>()
        val card = AnswerCard(question = AssistantQuestion("Which project?", listOf("withdo", "picpocket")))
        compose.setThemedContent { AssistantAbove(card, onUndo = {}, onPick = { picked += it }, onSignIn = {}) }
        compose.onAllNodesWithTag("assistant-option").assertCountEquals(2)
        compose.onNodeWithText("picpocket").performClick()
        compose.onNodeWithText("withdo").performClick()
        assertEquals(listOf("picpocket", "withdo"), picked)
    }

    @Test
    fun `the quiet line says what the local rules left out`() {
        compose.setThemedContent {
            AssistantAbove(receipts.copy(quiet = QuietNote.USED_UP), onUndo = {}, onPick = {}, onSignIn = {})
        }
        compose.onNodeWithTag("assistant-quiet").assertIsDisplayed()
        compose.onNodeWithText("used up", substring = true, ignoreCase = true).assertIsDisplayed()
    }

    @Test
    fun `while the model thinks Send is Stop and the text stays`() {
        var stopped = 0
        var sent = 0
        compose.mainClock.autoAdvance = false
        compose.setThemedContent {
            Composer(
                value = "What did Claude finish this week?",
                placeholder = "",
                notice = null,
                sending = true,
                onValueChange = {},
                onSend = { sent++ },
                onStop = { stopped++ },
                thinking = true,
            )
        }
        compose.onNodeWithTag("assistant-thinking").assertIsDisplayed()
        compose.onNodeWithText("Thinking…").assertIsDisplayed()
        compose.onNodeWithText("What did Claude finish this week?").assertIsDisplayed()
        compose.onNodeWithContentDescription("Stop").performClick()
        assertEquals(1 to 0, stopped to sent)
    }

    @Test
    fun `the model chip opens its sheet with today's turns`() {
        compose.setThemedContent {
            var open = androidx.compose.runtime.remember { androidx.compose.runtime.mutableStateOf(false) }
            ModelChip { open.value = true }
            if (open.value) ModelSheet(ModelState("official", ModelPhase.READY, 86, 100)) { open.value = false }
        }
        compose.onNodeWithTag("model-chip").performClick()
        compose.onNodeWithText("Available · 86 left today").assertIsDisplayed()
    }

    @Test
    fun `a touch outside closes the card and still reaches what it was for`() {
        var closed = 0
        var clicked = 0
        compose.setThemedContent {
            Box(Modifier.fillMaxSize().dismissOnTouch(enabled = true) { closed++ }) {
                Box(Modifier.size(100.dp).clickable { clicked++ }.testTag("page-row"))
            }
        }
        compose.onNodeWithTag("page-row").performClick()
        assertEquals(1 to 1, closed to clicked)
    }

    @Test
    fun `no card, no listening`() {
        var closed = 0
        compose.setThemedContent {
            Box(Modifier.fillMaxSize().dismissOnTouch(enabled = false) { closed++ }) {
                Box(Modifier.size(100.dp).clickable { }.testTag("page-row"))
            }
        }
        compose.onNodeWithTag("page-row").performClick()
        assertEquals(0, closed)
    }
}
