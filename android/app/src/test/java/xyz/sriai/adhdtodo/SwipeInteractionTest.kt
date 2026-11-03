package xyz.sriai.adhdtodo

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.Text
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.swipeLeft
import androidx.compose.ui.test.swipeRight
import androidx.compose.ui.unit.dp
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.testing.setThemedContent
import xyz.sriai.adhdtodo.ui.LocalSelection
import xyz.sriai.adhdtodo.ui.RowSwipes
import xyz.sriai.adhdtodo.ui.SelectionUi
import xyz.sriai.adhdtodo.ui.SwipeAction
import xyz.sriai.adhdtodo.ui.SwipeRow

/** A row's swipe (v43 P2): past half it runs on release, short of it nothing runs, and select mode does not swipe. */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class SwipeInteractionTest {
    @get:Rule
    val compose = createComposeRule()

    private var ran: SwipeAction? = null

    private fun row(swipes: RowSwipes, selecting: Boolean = false) {
        compose.setThemedContent {
            val selection = if (selecting) SelectionUi(emptySet(), emptyList(), {}, {}) else null
            CompositionLocalProvider(LocalSelection provides selection) {
                SwipeRow(swipes = swipes, onSwipe = { ran = it }, modifier = Modifier.testTag("row")) {
                    Text("Book dentist", Modifier.fillMaxWidth().height(48.dp))
                }
            }
        }
    }

    @Test
    fun `a long right swipe completes`() {
        row(RowSwipes(SwipeAction.Done, SwipeAction.Today))
        compose.onNodeWithTag("row").performTouchInput { swipeRight(startX = left + 4f, endX = right - 4f) }
        compose.waitForIdle()
        assertEquals(SwipeAction.Done, ran)
    }

    @Test
    fun `a long left swipe moves it`() {
        row(RowSwipes(SwipeAction.Done, SwipeAction.Tomorrow))
        compose.onNodeWithTag("row").performTouchInput { swipeLeft(startX = right - 4f, endX = left + 4f) }
        compose.waitForIdle()
        assertEquals(SwipeAction.Tomorrow, ran)
    }

    @Test
    fun `a short swipe springs back`() {
        row(RowSwipes(SwipeAction.Done, SwipeAction.Today))
        compose.onNodeWithTag("row").performTouchInput { swipeRight(startX = left + 4f, endX = left + width * 0.3f) }
        compose.waitForIdle()
        assertNull(ran)
    }

    @Test
    fun `an agents task does not swipe left`() {
        row(RowSwipes(SwipeAction.Done, null))
        compose.onNodeWithTag("row").performTouchInput { swipeLeft(startX = right - 4f, endX = left + 4f) }
        compose.waitForIdle()
        assertNull(ran)
    }

    @Test
    fun `rows in select mode do not swipe`() {
        row(RowSwipes(SwipeAction.Done, SwipeAction.Today), selecting = true)
        compose.onNodeWithTag("row").performTouchInput { swipeRight(startX = left + 4f, endX = right - 4f) }
        compose.waitForIdle()
        assertNull(ran)
    }
}
